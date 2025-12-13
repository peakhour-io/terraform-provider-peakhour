package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/onboard"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return fmt.Sprint([]string(*s)) }
func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	var outDir string
	var domainsFlag stringSliceFlag
	var allDomains bool
	var concurrency int

	flag.StringVar(&outDir, "out", "./out", "Output directory")
	flag.Var(&domainsFlag, "domain", "Domain to onboard (repeatable)")
	flag.BoolVar(&allDomains, "all-domains", false, "Onboard all domains visible to the API key")
	flag.IntVar(&concurrency, "concurrency", 4, "Max concurrent domain inventory requests")
	flag.Parse()

	apiKey := os.Getenv("PEAKHOUR_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "PEAKHOUR_API_KEY is required")
		os.Exit(2)
	}

	baseURL := os.Getenv("PEAKHOUR_BASE_URL")
	c := client.NewClient(apiKey, baseURL)

	domains, err := resolveDomains(c, allDomains, []string(domainsFlag))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	if concurrency < 1 {
		concurrency = 1
	}

	if err := writeDomains(context.Background(), c, outDir, domains, concurrency); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func resolveDomains(c *client.Client, allDomains bool, explicit []string) ([]string, error) {
	seen := map[string]struct{}{}
	for _, d := range explicit {
		if d == "" {
			continue
		}
		seen[d] = struct{}{}
	}

	if allDomains {
		names, err := c.ListDomains()
		if err != nil {
			return nil, fmt.Errorf("list domains: %w", err)
		}
		for _, n := range names {
			seen[n] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil, fmt.Errorf("no domains specified; use --domain or --all-domains")
	}

	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Strings(out)
	return out, nil
}

func writeDomains(ctx context.Context, c *client.Client, outDir string, domains []string, concurrency int) error {
	root := filepath.Join(outDir, "domains")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create output dir %q: %w", root, err)
	}

	domainCh := make(chan string)
	errCh := make(chan error, 1)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range domainCh {
				if err := writeDomain(ctx, c, outDir, domain); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
		}()
	}

	for _, domain := range domains {
		select {
		case domainCh <- domain:
		case err := <-errCh:
			close(domainCh)
			wg.Wait()
			return err
		}
	}
	close(domainCh)
	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func writeDomain(ctx context.Context, c *client.Client, outDir, domain string) error {
	targets, err := onboard.CollectDomainInventory(ctx, c, domain)
	if err != nil {
		return fmt.Errorf("%s: collect inventory: %w", domain, err)
	}

	dir := filepath.Join(outDir, "domains", domain)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%s: create dir %q: %w", domain, dir, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "provider.tf"), []byte(onboard.RenderProviderTF()), 0o644); err != nil {
		return fmt.Errorf("%s: write provider.tf: %w", domain, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "imports.tf"), []byte(onboard.RenderImportsTF(targets)), 0o644); err != nil {
		return fmt.Errorf("%s: write imports.tf: %w", domain, err)
	}
	return nil
}


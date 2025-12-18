# Image Transforms Guide

Named image transforms allow you to define reusable presets for image processing. Each preset specifies transformations like resizing, format conversion, watermarking, and color adjustments that are applied when images are requested.

## Resource Structure

```hcl
resource "peakhour_image_transform" "example" {
  domain = "images.example.com"
  name   = "thumbnail"
  config_json = jsonencode({
    w   = 300
    h   = 300
    fit = "crop"
    fm  = "WEBP"
    q   = 80
  })
}

# Commit changes to make them live
resource "peakhour_image_transform_commit" "example" {
  domain = "images.example.com"
  triggers = {
    thumbnail = peakhour_image_transform.example.id
  }
}
```

## Committing Changes

Image transform changes are staged until committed. Use `peakhour_image_transform_commit` to deploy changes:

```hcl
resource "peakhour_image_transform_commit" "all" {
  domain = "images.example.com"

  # Triggers commit when any transform changes
  triggers = {
    thumbnail = peakhour_image_transform.thumbnail.id
    hero      = peakhour_image_transform.hero.id
    product   = peakhour_image_transform.product.id
  }
}
```

## Configuration Reference

The config uses a **flat structure** where all parameters are at the top level. This matches the URL query parameter format used by the Image Optimization API.

### Size Options

Control image dimensions and cropping behavior.

| Field | Type | Description |
|-------|------|-------------|
| `w` | integer | Width in pixels |
| `h` | integer | Height in pixels |
| `fit` | string | Fit mode: `clip` (default), `crop`, `fill`, `scale`, `facearea` |
| `dpr` | number | Device pixel ratio multiplier |
| `crop` | string | Crop strategy: `centre`, `attention`, `entropy`, `edges`, `faces`, `objects`, `focalpoint`, or directional (`top`, `bottom`, `left`, `right`) |

**Fit Modes:**
- `clip` (default) - Resize to fit within dimensions, preserving aspect ratio
- `crop` - Crop to exact dimensions
- `fill` - Fill dimensions, adding padding if needed
- `scale` - Stretch to exact dimensions (may distort)
- `facearea` - Crop centered on detected faces

### Format Options

Control output format and quality.

| Field | Type | Description |
|-------|------|-------------|
| `fm` | string | Output format: `GIF`, `JPEG`, `PNG`, `WEBP`, `SVG`, `AVIF`, `JXL` |
| `q` | int/string | Quality 1-100, or `auto`, `auto:high`, `auto:med`, `auto:low` |
| `chromasub` | string | Chroma subsampling: `4:4:4`, `4:2:2`, `4:2:0` |
| `lossless` | boolean | Use lossless compression |
| `cs` | string | Color space: `RGB`, `RGBA`, `RGB16`, `GREY16`, `P`, `L`, `CMYK` |

### Fill Options

Control background and fill colors.

| Field | Type | Description |
|-------|------|-------------|
| `fill` | string | Fill mode |
| `fill_color` | string | Fill color (hex, e.g., `FFFFFF`) |
| `bg` | string | Background color (hex) |

### Mask Options

Apply shape masks to images.

| Field | Type | Description |
|-------|------|-------------|
| `mask` | string | Mask type: `corners`, `ellipse` |
| `corner_radius` | number | Corner radius for `corners` mask |
| `mask_bg` | string | Background color for masked areas |

### Watermark Options

Add watermark overlays.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `wm_url` | string | - | URL to watermark image |
| `wm_pos` | string | `center` | Position: `center`, `northwest`, `northeast`, `southwest`, `southeast`, `north`, `south`, `east`, `west` |
| `wm_opacity` | number | 0.5 | Opacity 0-1 |
| `wm_pad` | integer | - | Padding from edge |
| `wm_x` | integer | - | X offset |
| `wm_y` | integer | - | Y offset |
| `wm_scale` | number | - | Scale factor |
| `wm_width` | integer | - | Width constraint |
| `wm_height` | integer | - | Height constraint |
| `wm_fit` | string | - | Fit mode: `clip`, `crop`, `fill`, `scale`, `max` |
| `wm_tile` | string | - | Tiling: `grid`, `x`, `y`, `both` |
| `wm_rot` | number | - | Rotation (-360 to 360 degrees) |
| `wm_if_min_width` | integer | - | Only apply if image width >= value |
| `wm_if_min_height` | integer | - | Only apply if image height >= value |

**Multiple Watermark Layers:**

Use indexed prefixes for additional watermarks:

```hcl
config_json = jsonencode({
  w          = 800
  h          = 600
  fit        = "crop"
  wm_url     = "https://example.com/logo.png"
  wm_pos     = "northwest"
  wm_opacity = 1
  # Additional layer
  wm1_url   = "https://example.com/badge.png"
  wm1_pos   = "southwest"
  wm1_scale = 40
  wm1_y     = -15
  fm        = "JPEG"
})
```

### Blur Options

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| `blur` | number | 0.1-10.0 | Blur intensity |

### Sharpen Options

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| `usm` | number | 0.5-5.0 | Unsharp mask intensity |
| `usmrad` | number | 1.0-10.0 | Unsharp mask radius |
| `sharp` | number | 1.0-100.0 | Simple sharpening |

### Color Options

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| `brightness` | number | -100 to 100 | Brightness adjustment |
| `contrast` | number | -100 to 100 | Contrast adjustment |
| `saturation` | number | -100 to 100 | Saturation adjustment |
| `gamma` | number | 0.1-5.0 | Gamma correction |

### Tone Options

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| `hue` | number | 0-359 | Hue rotation (degrees) |
| `vib` | number | -100 to 100 | Vibrance |
| `high` | number | -100 to 0 | Highlights adjustment |
| `shad` | number | 0-100 | Shadows adjustment |

### Style Options

| Field | Type | Description |
|-------|------|-------------|
| `sepia` | number | Sepia intensity (0-100) |
| `grayscale` | boolean | Convert to grayscale |
| `monochrome` | string | Monochrome with color tint |
| `duotone` | string | Duotone effect (`color1,color2`) |

### Geometry Options

| Field | Type | Description |
|-------|------|-------------|
| `flip` | string | Flip: `h` (horizontal), `v` (vertical), `hv` (both) |
| `rot` | number | Rotation 0-360 degrees |
| `pad` | integer | Padding (all sides) |
| `pad_top`, `pad_right`, `pad_bottom`, `pad_left` | integer | Directional padding |
| `border` | integer | Border width |
| `border_color` | string | Border color (hex) |
| `border_radius` | integer | Border corner radius |
| `border_radius_inner` | integer | Inner border radius |

### Trim Options

| Field | Type | Description |
|-------|------|-------------|
| `trim` | string/boolean | Trim mode: `color`, `auto`, or `true` |
| `trimcolor` | string | Color to trim (hex) |
| `trimtol` | number | Tolerance for color matching |
| `trimpad` | integer | Padding to preserve after trimming |

### Text Options

| Field | Type | Description |
|-------|------|-------------|
| `txt` | string | Text content |
| `txt_size` | integer | Font size |
| `txt_font` | string | Font name |
| `txt_color` | string | Text color |
| `txt_align` | string | Alignment (`left,top`, `center,middle`, etc.) |
| `txt_x`, `txt_y` | integer | Position in pixels |
| `txt_width` | integer | Maximum text width |
| `txt_fit` | string | Fitting mode: `max` |
| `txt_clip` | string | Clipping: `start`, `middle`, `end`, `ellipsis` |
| `txt_pad` | integer | Padding around text |
| `txt_lead` | integer | Line height (leading) |
| `txt_track` | integer | Letter spacing (tracking) |
| `txt_shad` | integer | Text shadow |
| `txt64` | string | Base64-encoded text |
| `txt_font64` | string | Base64-encoded font URL |

### Auto Options

| Field | Type | Description |
|-------|------|-------------|
| `auto` | string | Comma-separated: `optimize`, `format`, `ssim` |

Or as an object:

| Field | Type | Description |
|-------|------|-------------|
| `auto.optimize` | boolean | Enable automatic optimization |
| `auto.format` | boolean | Enable automatic format selection |
| `auto.ssim` | boolean | Enable SSIM-based quality optimization |

### Client Hints (ch)

| Field | Type | Description |
|-------|------|-------------|
| `ch` | string | Comma-separated: `width`, `dpr`, `savedata` |

Or as an object:

| Field | Type | Description |
|-------|------|-------------|
| `ch.width` | boolean | Respond to width client hints |
| `ch.dpr` | boolean | Respond to DPR client hints |
| `ch.savedata` | boolean | Respond to save-data hint |

---

## Complete Examples

### Basic Thumbnail

```hcl
resource "peakhour_image_transform" "thumbnail" {
  domain = peakhour_domain.images.name
  name   = "thumbnail"
  config_json = jsonencode({
    w   = 300
    h   = 300
    fit = "crop"
    fm  = "WEBP"
    q   = 80
  })
}
```

### Hero Image with Auto Optimization

```hcl
resource "peakhour_image_transform" "hero" {
  domain = peakhour_domain.images.name
  name   = "hero"
  config_json = jsonencode({
    w    = 1920
    h    = 1080
    fit  = "clip"
    fm   = "AVIF"
    q    = "auto:high"
    auto = "optimize"
  })
}
```

### Product Image with Watermark

```hcl
resource "peakhour_image_transform" "product" {
  domain = peakhour_domain.images.name
  name   = "product-watermarked"
  config_json = jsonencode({
    w          = 460
    h          = 307
    fit        = "crop"
    wm_url     = "https://example.com/logo.png"
    wm_opacity = 0.8
    wm_pos     = "southeast"
    fm         = "WEBP"
    q          = 85
  })
}
```

### Multiple Watermark Layers

```hcl
resource "peakhour_image_transform" "dual_watermark" {
  domain = peakhour_domain.images.name
  name   = "dual-watermark"
  config_json = jsonencode({
    w          = 800
    h          = 600
    fit        = "crop"
    wm_url     = "https://example.com/main-logo.png"
    wm_pos     = "northwest"
    wm_opacity = 1
    # Additional watermark layer
    wm1_url   = "https://example.com/badge.png"
    wm1_pos   = "southwest"
    wm1_scale = 40
    wm1_y     = -15
    fm        = "JPEG"
  })
}
```

### Grayscale with Border

```hcl
resource "peakhour_image_transform" "grayscale_framed" {
  domain = peakhour_domain.images.name
  name   = "grayscale-framed"
  config_json = jsonencode({
    w             = 500
    h             = 500
    fit           = "fill"
    grayscale     = true
    border        = 5
    border_color  = "333333"
    border_radius = 10
    bg            = "FFFFFF"
  })
}
```

### Face-Aware Avatar

```hcl
resource "peakhour_image_transform" "avatar" {
  domain = peakhour_domain.images.name
  name   = "profile-avatar"
  config_json = jsonencode({
    w    = 150
    h    = 150
    fit  = "crop"
    crop = "faces"
    mask = "ellipse"
    fm   = "PNG"
  })
}
```

### Blurred Background

```hcl
resource "peakhour_image_transform" "blurred" {
  domain = peakhour_domain.images.name
  name   = "blurred-bg"
  config_json = jsonencode({
    blur = 5.0
    fm   = "JPEG"
    q    = 60
  })
}
```

### Color Adjustments

```hcl
resource "peakhour_image_transform" "vibrant" {
  domain = peakhour_domain.images.name
  name   = "vibrant"
  config_json = jsonencode({
    brightness = 10
    contrast   = 15
    saturation = 25
    vib        = 30
    sharp      = 15
  })
}
```

### Trim Product Images

```hcl
resource "peakhour_image_transform" "trimmed" {
  domain = peakhour_domain.images.name
  name   = "product-trimmed"
  config_json = jsonencode({
    trim      = "color"
    trimcolor = "ffffff"
    trimtol   = 15
    trimpad   = 20
    fm        = "WEBP"
    q         = 85
  })
}
```

---

## Importing Existing Transforms

To import an existing image transform preset:

```hcl
import {
  to = peakhour_image_transform.example
  id = "images.example.com/550e8400-e29b-41d4-a716-446655440000"
}
```

The import ID format is `domain/uuid`.

---

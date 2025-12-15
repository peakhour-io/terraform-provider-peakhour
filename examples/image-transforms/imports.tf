# Import blocks for existing infrastructure
# Uncomment the imports for resources that already exist in your Peakhour account.
# After importing, you can remove these blocks.

# Import existing domains
import {
  to = peakhour_domain.example
  id = "example.com"
}

import {
  to = peakhour_domain.images
  id = "images.example.com"
}

# Import existing domain plans
import {
  to = peakhour_domain_plan.example
  id = "example.com"
}

import {
  to = peakhour_domain_plan.images
  id = "images.example.com"
}

# Import existing reverse proxy services
import {
  to = peakhour_reverse_proxy_service.example
  id = "example.com"
}

import {
  to = peakhour_reverse_proxy_service.images
  id = "images.example.com"
}

# Import existing origin pools
import {
  to = peakhour_origin_pool.backend
  id = "example.com/origins/production"
}

# Import existing image transforms (replace <uuid> with actual UUIDs)
# You can find UUIDs via: curl -H "Authorization: Bearer $PEAKHOUR_API_KEY" \
#   https://console.peakhour.io/api/v1/domains/images.example.com/image-transforms
#
import {
  to = peakhour_image_transform.thumbnail
  id = "images.example.com/21ad41ff-fadc-4a28-a4e3-06df17615614"
}

import {
  to = peakhour_image_transform.hero
  id = "images.example.com/cf6344dd-39ff-48d4-8bfa-280d6e690115"
}

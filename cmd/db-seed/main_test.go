package main

import "testing"

func TestDefaultProductSeeds_HasMultipleVariantProducts(t *testing.T) {
	seeds := defaultProductSeeds()

	variantCounts := make([]int, 0, len(seeds))
	for _, seed := range seeds {
		if len(seed.variants) > 1 {
			variantCounts = append(variantCounts, len(seed.variants))
		}
	}

	if len(variantCounts) < 2 {
		t.Fatalf("expected at least two products with more than one variant, got %d", len(variantCounts))
	}

	hasTwo := false
	hasThree := false
	for _, c := range variantCounts {
		if c == 2 {
			hasTwo = true
		}
		if c == 3 {
			hasThree = true
		}
	}

	if !hasTwo || !hasThree {
		t.Fatalf("expected one product with 2 variants and one product with 3 variants, got %v", variantCounts)
	}
}

func TestDefaultProductSeeds_VariantSKUsAreUnique(t *testing.T) {
	seeds := defaultProductSeeds()

	seen := make(map[string]struct{})
	for _, seed := range seeds {
		for _, variant := range seed.variants {
			if variant.sku == "" {
				t.Fatalf("empty variant sku found in product %q", seed.name)
			}
			if _, exists := seen[variant.sku]; exists {
				t.Fatalf("duplicate variant sku found: %s", variant.sku)
			}
			seen[variant.sku] = struct{}{}
		}
	}
}

func TestVariantSeedRunMessage(t *testing.T) {
	tests := []struct {
		name       string
		product    string
		configured int
		created    int
		updated    int
		want       string
	}{
		{
			name:       "variant product summary",
			product:    "Premium T-Shirt",
			configured: 3,
			created:    1,
			updated:    2,
			want:       "      📊 Variant seed result [Premium T-Shirt]: created=1 updated=2 configured=3",
		},
		{
			name:       "simple product summary",
			product:    "Classic Cap",
			configured: 0,
			created:    0,
			updated:    0,
			want:       "      📊 Variant seed result [Classic Cap]: created=0 updated=0 (no configured variants)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := variantSeedRunMessage(tc.product, tc.configured, tc.created, tc.updated)
			if got != tc.want {
				t.Fatalf("unexpected message\nwant: %q\ngot:  %q", tc.want, got)
			}
		})
	}
}

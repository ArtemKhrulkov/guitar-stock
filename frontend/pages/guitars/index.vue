<template>
  <div class="catalog-page">
    <div class="catalog-header mb-4">
      <v-container>
        <div class="d-flex align-center justify-space-between flex-wrap ga-4 mb-4">
          <div>
            <h1 class="text-h4 font-weight-bold mb-1">Guitar Catalog</h1>
            <p class="text-body-2 text-medium-emphasis">{{ total }} guitars found</p>
          </div>
          <div class="d-flex align-center ga-2">
            <v-select
              v-model="sortValue"
              :items="sortOptions"
              variant="outlined"
              density="compact"
              hide-details
              prepend-inner-icon="mdi-sort"
              class="sort-select"
              style="max-width: 180px"
              @update:model-value="onSortChange"
            />
            <v-btn-toggle v-model="viewMode" mandatory variant="outlined" color="primary">
              <v-btn value="grid" size="small" aria-label="Grid view">
                <IconifyIcon icon="mdi-view-grid" />
              </v-btn>
              <v-btn value="list" size="small" aria-label="List view">
                <IconifyIcon icon="mdi-view-list" />
              </v-btn>
            </v-btn-toggle>
          </div>
        </div>
      </v-container>
    </div>

    <v-container fluid class="pa-6 pt-0">
      <v-row>
        <v-col cols="12" md="3">
          <div class="filters-sidebar">
            <GuitarFilters
              v-model:selected-brands="selectedBrands"
              v-model:selected-type="selectedType"
              v-model:selected-sort="selectedSort"
              v-model:min-price="minPrice"
              v-model:max-price="maxPrice"
              v-model:in-stock="inStock"
              v-model:search-query="searchQuery"
              :brands="brands"
              :loading="filtersLoading"
              @apply-filters="applyFilters"
              @clear-filters="clearFilters"
            />
          </div>
        </v-col>

        <v-col cols="12" md="9">
          <div class="catalog-toolbar">
            <v-text-field
              v-model="searchQuery"
              placeholder="Search by model or history..."
              prepend-inner-icon="mdi-magnify"
              variant="outlined"
              density="compact"
              hide-details
              class="search-field"
              clearable
              @input="onSearchInput"
            />

            <div v-if="hasActiveFilters" class="active-filters">
              <v-chip
                v-for="brand in selectedBrandChips"
                :key="brand.id"
                closable
                size="small"
                color="secondary"
                variant="tonal"
                class="mr-2 mb-2"
                @click:close="removeBrand(brand.id)"
              >
                <IconifyIcon icon="mdi-factory" size="14" class="mr-1" />
                {{ brand.name }}
              </v-chip>
              <v-chip
                v-if="selectedType"
                closable
                size="small"
                color="primary"
                variant="tonal"
                class="mr-2 mb-2"
                @click:close="selectedType = ''"
              >
                <IconifyIcon icon="mdi-guitar-electric" size="14" class="mr-1" />
                {{ selectedType }}
              </v-chip>
              <v-chip
                v-if="priceRangeLabel"
                closable
                size="small"
                color="warning"
                variant="tonal"
                class="mr-2 mb-2"
                @click:close="clearPriceRange"
              >
                <IconifyIcon icon="mdi-currency-rub" size="14" class="mr-1" />
                {{ priceRangeLabel }}
              </v-chip>
              <v-chip
                v-if="inStock"
                closable
                size="small"
                color="success"
                variant="tonal"
                class="mr-2 mb-2"
                @click:close="inStock = undefined"
              >
                <IconifyIcon icon="mdi-package-variant-closed" size="14" class="mr-1" />
                In Stock
              </v-chip>
              <v-btn
                v-if="hasActiveFilters"
                size="x-small"
                variant="text"
                color="error"
                @click="clearFilters"
              >
                Clear all
              </v-btn>
            </div>
          </div>

          <div v-if="loading" :class="viewMode === 'grid' ? 'guitars-grid' : 'guitars-list'">
            <v-skeleton-loader
              v-for="n in 6"
              :key="n"
              :type="viewMode === 'grid' ? 'card' : 'list-item-avatar-three-line'"
            />
          </div>

          <v-card v-else-if="guitars.length === 0" class="text-center pa-12 empty-state">
            <IconifyIcon icon="mdi-guitar-electric" size="80" color="grey" class="mb-4" />
            <h3 class="text-h5 mb-2">No guitars found</h3>
            <p class="text-body-2 text-medium-emphasis mb-4">
              Try adjusting your filters or search query
            </p>
            <v-btn color="primary" variant="tonal" @click="clearFilters">
              <IconifyIcon icon="mdi-refresh" class="mr-2" />
              Clear Filters
            </v-btn>
          </v-card>

          <div v-else :class="viewMode === 'grid' ? 'guitars-grid' : 'guitars-list'">
            <GuitarCard v-for="guitar in guitars" :key="guitar.id" :guitar="guitar" />
          </div>

          <div v-if="totalPages > 1" class="pagination-wrapper mt-8">
            <v-pagination
              v-model="currentPage"
              :length="totalPages"
              :total-visible="7"
              rounded="lg"
              @update:model-value="changePage"
            />
          </div>
        </v-col>
      </v-row>
    </v-container>
  </div>
</template>

<script setup lang="ts">
const route = useRoute();
const router = useRouter();

const { guitars, total, loading, fetchGuitars } = useGuitars();
const { brands, loading: filtersLoading, fetchBrands } = useBrands();

const selectedBrands = ref<string[]>([]);
const selectedType = ref<string>('');
const selectedSort = ref<string>('newest');
const minPrice = ref<number | undefined>(undefined);
const maxPrice = ref<number | undefined>(undefined);
const inStock = ref<boolean | undefined>(undefined);
const searchQuery = ref<string>('');
const currentPage = ref<number>(1);
const viewMode = ref<'grid' | 'list'>('grid');
const itemsPerPage = 12;

const sortOptions = [
  { title: 'Newest First', value: 'newest' },
  { title: 'Oldest First', value: 'oldest' },
  { title: 'A-Z', value: 'model_asc' },
  { title: 'Z-A', value: 'model_desc' },
];

const sortValue = computed({
  get: () => selectedSort.value,
  set: (val) => {
    selectedSort.value = val || 'newest';
  },
});

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

const hasActiveFilters = computed(() => {
  return (
    selectedBrands.value.length > 0 ||
    selectedType.value ||
    minPrice.value !== undefined ||
    maxPrice.value !== undefined ||
    inStock.value !== undefined ||
    searchQuery.value
  );
});

const totalPages = computed(() => Math.ceil(total.value / itemsPerPage));

const selectedBrandChips = computed(() => {
  return brands.value
    .filter((b) => selectedBrands.value.includes(b.id))
    .map((b) => ({ id: b.id, name: b.name }));
});

const priceRangeLabel = computed(() => {
  if (minPrice.value && maxPrice.value) {
    return `${formatPrice(minPrice.value)} - ${formatPrice(maxPrice.value)}`;
  }
  if (minPrice.value) {
    return `от ${formatPrice(minPrice.value)}`;
  }
  if (maxPrice.value) {
    return `до ${formatPrice(maxPrice.value)}`;
  }
  return '';
});

const formatPrice = (price: number) => {
  if (price >= 1000) {
    return `${(price / 1000).toFixed(0)}K`;
  }
  return price.toString();
};

const getSortParams = (sort: string) => {
  switch (sort) {
    case 'newest':
      return { sort: 'newest' as const, dir: 'desc' as const };
    case 'oldest':
      return { sort: 'newest' as const, dir: 'asc' as const };
    case 'model_asc':
      return { sort: 'model' as const, dir: 'asc' as const };
    case 'model_desc':
      return { sort: 'model' as const, dir: 'desc' as const };
    default:
      return { sort: 'newest' as const, dir: 'desc' as const };
  }
};

const applyFilters = async () => {
  currentPage.value = 1;
  updateURL();
  const sortParams = getSortParams(selectedSort.value);
  await fetchGuitars({
    brands: selectedBrands.value.length > 0 ? selectedBrands.value : undefined,
    type: (selectedType.value as 'electric' | 'acoustic' | 'bass') || undefined,
    search: searchQuery.value || undefined,
    min_price: minPrice.value,
    max_price: maxPrice.value,
    in_stock: inStock.value,
    sort: sortParams.sort,
    dir: sortParams.dir,
    page: currentPage.value,
    limit: itemsPerPage,
  });
};

const onSearchInput = () => {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer);
  }
  searchDebounceTimer = setTimeout(() => {
    applyFilters();
  }, 500);
};

const onSortChange = () => {
  applyFilters();
};

const clearFilters = async () => {
  selectedBrands.value = [];
  selectedType.value = '';
  minPrice.value = undefined;
  maxPrice.value = undefined;
  inStock.value = undefined;
  searchQuery.value = '';
  currentPage.value = 1;
  await applyFilters();
};

const clearPriceRange = () => {
  minPrice.value = undefined;
  maxPrice.value = undefined;
};

const removeBrand = (brandId: string) => {
  selectedBrands.value = selectedBrands.value.filter((b) => b !== brandId);
};

const changePage = async (page: number) => {
  currentPage.value = page;
  updateURL();
  const sortParams = getSortParams(selectedSort.value);
  await fetchGuitars({
    brands: selectedBrands.value.length > 0 ? selectedBrands.value : undefined,
    type: (selectedType.value as 'electric' | 'acoustic' | 'bass') || undefined,
    search: searchQuery.value || undefined,
    min_price: minPrice.value,
    max_price: maxPrice.value,
    in_stock: inStock.value,
    sort: sortParams.sort,
    dir: sortParams.dir,
    page: currentPage.value,
    limit: itemsPerPage,
  });
  window.scrollTo({ top: 0, behavior: 'smooth' });
};

const updateURL = () => {
  const query: Record<string, string> = {};
  if (selectedBrands.value.length === 1) {
    query.brand = selectedBrands.value[0];
  } else if (selectedBrands.value.length > 1) {
    query.brands = selectedBrands.value.join(',');
  }
  if (selectedType.value) query.type = selectedType.value;
  if (searchQuery.value) query.search = searchQuery.value;
  if (minPrice.value) query.min_price = minPrice.value.toString();
  if (maxPrice.value) query.max_price = maxPrice.value.toString();
  if (inStock.value) query.in_stock = 'true';
  if (selectedSort.value !== 'newest') query.sort = selectedSort.value;
  if (currentPage.value > 1) query.page = currentPage.value.toString();
  router.replace({ query });
};

useHead({
  title: 'Guitar Catalog',
});

await fetchBrands();

const parseQueryParams = () => {
  if (route.query.brand) {
    selectedBrands.value = Array.isArray(route.query.brand)
      ? route.query.brand as string[]
      : [route.query.brand as string];
  }
  if (route.query.brands) {
    selectedBrands.value = (route.query.brands as string).split(',');
  }
  if (route.query.type) {
    selectedType.value = route.query.type as string;
  }
  if (route.query.search) {
    searchQuery.value = route.query.search as string;
  }
  if (route.query.min_price) {
    minPrice.value = parseInt(route.query.min_price as string);
  }
  if (route.query.max_price) {
    maxPrice.value = parseInt(route.query.max_price as string);
  }
  if (route.query.in_stock) {
    inStock.value = route.query.in_stock === 'true';
  }
  if (route.query.sort) {
    selectedSort.value = route.query.sort as string;
  }
  if (route.query.page) {
    currentPage.value = parseInt(route.query.page as string);
  }
};

parseQueryParams();

const sortParams = getSortParams(selectedSort.value);
await fetchGuitars({
  brands: selectedBrands.value.length > 0 ? selectedBrands.value : undefined,
  type: (selectedType.value as 'electric' | 'acoustic' | 'bass') || undefined,
  search: searchQuery.value || undefined,
  min_price: minPrice.value,
  max_price: maxPrice.value,
  in_stock: inStock.value,
  sort: sortParams.sort,
  dir: sortParams.dir,
  page: currentPage.value,
  limit: itemsPerPage,
});

watch(
  () => route.query,
  async () => {
    parseQueryParams();
    const sortParams = getSortParams(selectedSort.value);
    await fetchGuitars({
      brands: selectedBrands.value.length > 0 ? selectedBrands.value : undefined,
      type: (selectedType.value as 'electric' | 'acoustic' | 'bass') || undefined,
      search: searchQuery.value || undefined,
      min_price: minPrice.value,
      max_price: maxPrice.value,
      in_stock: inStock.value,
      sort: sortParams.sort,
      dir: sortParams.dir,
      page: currentPage.value,
      limit: itemsPerPage,
    });
  },
);
</script>

<style scoped>
.catalog-page {
  min-height: 100%;
  background: linear-gradient(
    180deg,
    rgba(var(--v-theme-background)) 0%,
    rgba(var(--v-theme-surface)) 100%
  );
}

.catalog-header {
  background: rgba(var(--v-theme-surface));
  padding: 24px 0;
  border-bottom: 1px solid rgba(var(--v-theme-primary), 0.1);
}

.filters-sidebar {
  position: sticky;
  top: 80px;
}

.catalog-toolbar {
  margin-bottom: 24px;
}

.search-field {
  max-width: 400px;
}

.sort-select {
  min-width: 160px;
}

.active-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  margin-top: 12px;
}

.guitars-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
}

@media (max-width: 1200px) {
  .guitars-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 600px) {
  .guitars-grid {
    grid-template-columns: 1fr;
  }
}

.guitars-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.empty-state {
  padding: 60px 24px;
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
}
</style>

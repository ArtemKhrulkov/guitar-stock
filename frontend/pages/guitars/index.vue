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
              v-model:selected-brand="selectedBrand"
              v-model:selected-type="selectedType"
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
                v-if="selectedBrand"
                closable
                size="small"
                color="secondary"
                variant="tonal"
                class="mr-2 mb-2"
                @click:close="selectedBrand = ''"
              >
                <IconifyIcon icon="mdi-factory" size="14" class="mr-1" />
                {{ getBrandName(selectedBrand) }}
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

const { guitars, total, loading, fetchGuitars } = useGuitars();
const { brands, loading: filtersLoading, fetchBrands } = useBrands();

const selectedBrand = ref<string>('');
const selectedType = ref<'electric' | 'acoustic' | 'bass' | ''>('');
const searchQuery = ref<string>('');
const currentPage = ref<number>(1);
const viewMode = ref<'grid' | 'list'>('grid');
const itemsPerPage = 12;

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

const hasActiveFilters = computed(() => {
  return selectedBrand.value || selectedType.value || searchQuery.value;
});

const totalPages = computed(() => Math.ceil(total.value / itemsPerPage));

const getBrandName = (brandId: string) => {
  const brand = brands.value.find((b) => b.id === brandId);
  return brand?.name || brandId;
};

const applyFilters = async () => {
  currentPage.value = 1;
  await fetchGuitars({
    brand: selectedBrand.value || undefined,
    type: selectedType.value || undefined,
    search: searchQuery.value || undefined,
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

const clearFilters = async () => {
  selectedBrand.value = '';
  selectedType.value = '';
  searchQuery.value = '';
  currentPage.value = 1;
  await applyFilters();
};

const changePage = async (page: number) => {
  currentPage.value = page;
  await fetchGuitars({
    brand: selectedBrand.value || undefined,
    type: selectedType.value || undefined,
    search: searchQuery.value || undefined,
    page: currentPage.value,
    limit: itemsPerPage,
  });
  window.scrollTo({ top: 0, behavior: 'smooth' });
};

useHead({
  title: 'Guitar Catalog',
});

await fetchBrands();

if (route.query.search) {
  searchQuery.value = route.query.search as string;
}
if (route.query.brand) {
  selectedBrand.value = route.query.brand as string;
}
if (route.query.type) {
  selectedType.value = route.query.type as 'electric' | 'acoustic' | 'bass' | '';
}
if (route.query.page) {
  currentPage.value = parseInt(route.query.page as string);
}

await applyFilters();

watch(
  () => route.query,
  async (query) => {
    if (query.search !== undefined) {
      searchQuery.value = query.search as string;
    }
    if (query.brand !== undefined) {
      selectedBrand.value = query.brand as string;
    }
    if (query.type !== undefined) {
      selectedType.value = query.type as 'electric' | 'acoustic' | 'bass' | '';
    }
    if (query.page !== undefined) {
      currentPage.value = parseInt(query.page as string);
    }
    await applyFilters();
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

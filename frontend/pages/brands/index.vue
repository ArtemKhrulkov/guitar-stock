<template>
  <div class="brands-page">
    <div class="brands-header mb-4">
      <v-container>
        <div class="d-flex align-center justify-space-between flex-wrap ga-4">
          <div>
            <h1 class="text-h4 font-weight-bold mb-1">Guitar Brands</h1>
            <p class="text-body-2 text-medium-emphasis">
              Explore {{ brands.length }} world-renowned manufacturers
            </p>
          </div>
          <v-text-field
            v-model="searchQuery"
            placeholder="Search brands..."
            prepend-inner-icon="mdi-magnify"
            variant="outlined"
            density="compact"
            hide-details
            class="search-field"
            clearable
          />
        </div>
      </v-container>
    </div>

    <v-container fluid class="pa-6 pt-0">
      <div v-if="loading" class="brands-grid">
        <v-skeleton-loader v-for="n in 12" :key="n" type="card" />
      </div>

      <v-card v-else-if="filteredBrands.length === 0" class="text-center pa-12 empty-state">
        <IconifyIcon icon="mdi-factory" size="80" color="grey" class="mb-4" />
        <h3 class="text-h5 mb-2">No brands found</h3>
        <p class="text-body-2 text-medium-emphasis mb-4">Try adjusting your search query</p>
        <v-btn color="primary" variant="tonal" @click="searchQuery = ''">
          <IconifyIcon icon="mdi-refresh" class="mr-2" />
          Clear Search
        </v-btn>
      </v-card>

      <div v-else class="brands-grid">
        <v-card
          v-for="brand in filteredBrands"
          :key="brand.id"
          :to="`/brands/${brand.id}`"
          class="brand-card"
        >
          <v-card-text class="text-center pa-6">
            <div class="brand-logo-wrapper mb-4">
              <v-avatar size="100" color="primary" class="brand-avatar">
                <NuxtImg
                  v-if="brand.logo_url"
                  :src="brand.logo_url"
                  :alt="`${brand.name} logo`"
                  width="100"
                  height="100"
                  loading="lazy"
                  format="webp"
                  quality="80"
                  class="brand-logo"
                />
                <IconifyIcon v-else icon="mdi-guitar-electric" size="48" />
              </v-avatar>
            </div>

            <h3 class="text-h6 font-weight-bold mb-2">{{ brand.name }}</h3>

            <div class="brand-meta mb-3">
              <v-chip size="x-small" color="secondary" variant="tonal" class="mr-2">
                <IconifyIcon icon="mdi-earth" size="12" class="mr-1" />
                {{ brand.country }}
              </v-chip>
              <v-chip v-if="brand.founded_year" size="x-small" variant="outlined">
                <IconifyIcon icon="mdi-calendar" size="12" class="mr-1" />
                Est. {{ brand.founded_year }}
              </v-chip>
            </div>

            <p v-if="brand.description" class="text-caption text-medium-emphasis brand-description">
              {{ truncateDescription(brand.description) }}
            </p>

            <v-btn variant="tonal" color="primary" size="small" class="mt-4">
              View Guitars
              <IconifyIcon icon="mdi-arrow-right" size="16" class="ml-1" />
            </v-btn>
          </v-card-text>
        </v-card>
      </div>
    </v-container>
  </div>
</template>

<script setup lang="ts">
const { brands, loading, fetchBrands } = useBrands();

const searchQuery = ref('');

const filteredBrands = computed(() => {
  if (!searchQuery.value) return brands.value;
  const query = searchQuery.value.toLowerCase();
  return brands.value.filter(
    (brand) =>
      brand.name.toLowerCase().includes(query) || brand.country.toLowerCase().includes(query),
  );
});

const truncateDescription = (description: string) => {
  if (description.length <= 100) return description;
  return description.substring(0, 100) + '...';
};

useHead({
  title: 'Guitar Brands',
});

await fetchBrands();
</script>

<style scoped>
.brands-page {
  min-height: 100%;
  background: linear-gradient(
    180deg,
    rgba(var(--v-theme-background)) 0%,
    rgba(var(--v-theme-surface)) 100%
  );
}

.brands-header {
  background: rgba(var(--v-theme-surface));
  padding: 24px 0;
  border-bottom: 1px solid rgba(var(--v-theme-primary), 0.1);
}

.search-field {
  max-width: 300px;
}

.brands-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

@media (max-width: 1200px) {
  .brands-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 900px) {
  .brands-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 600px) {
  .brands-grid {
    grid-template-columns: 1fr;
  }
}

.brand-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
  height: 100%;
}

.brand-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 16px 32px rgba(0, 0, 0, 0.2) !important;
}

.brand-logo-wrapper {
  display: flex;
  justify-content: center;
}

.brand-avatar {
  border: 4px solid rgba(var(--v-theme-primary), 0.2);
  transition: all 0.3s ease;
  overflow: hidden;
}

.brand-card:hover .brand-avatar {
  border-color: rgba(var(--v-theme-primary), 0.5);
  transform: scale(1.05);
}

.brand-logo {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.brand-meta {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
}

.brand-description {
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.empty-state {
  padding: 60px 24px;
}
</style>

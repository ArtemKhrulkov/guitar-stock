<template>
  <div class="detail-page">
    <v-container fluid class="pa-6">
      <div v-if="loading" class="loading-state">
        <v-skeleton-loader type="article" />
      </div>

      <v-card v-else-if="error" color="error" class="pa-8 text-center" role="alert">
        <IconifyIcon icon="mdi-alert-circle" size="64" class="mb-4" />
        <h2 class="text-h4 mb-4">Error Loading Guitar</h2>
        <p class="mb-4">{{ error }}</p>
        <v-btn to="/guitars" color="white" variant="flat" size="large">
          <IconifyIcon icon="mdi-arrow-left" class="mr-2" />
          Back to Catalog
        </v-btn>
      </v-card>

      <template v-else-if="guitar">
        <v-breadcrumbs :items="breadcrumbs" class="px-0 mb-6" aria-label="Breadcrumb navigation">
          <template #divider>
            <IconifyIcon icon="mdi-chevron-right" />
          </template>
        </v-breadcrumbs>

        <v-row>
          <v-col cols="12" lg="5">
            <div class="image-gallery">
              <v-card class="main-image-card" elevation="4">
                <div class="main-image-wrapper">
                  <NuxtImg
                    v-if="
                      guitar.image_url && !guitar.image_url.startsWith('https://via.placeholder')
                    "
                    :src="guitar.image_url"
                    :alt="`${guitar.brand?.name} ${guitar.model}`"
                    width="800"
                    height="600"
                    loading="eager"
                    format="webp"
                    quality="85"
                    class="main-image"
                  />
                  <div v-else class="image-placeholder">
                    <IconifyIcon icon="mdi-guitar-electric" size="80" />
                    <span class="mt-4 text-h6">{{ guitar.brand?.name }} {{ guitar.model }}</span>
                  </div>
                </div>
              </v-card>

              <div class="type-badge-wrapper mt-4">
                <v-chip :color="getTypeColor(guitar.guitar_type)" size="large" class="px-6">
                  <IconifyIcon :icon="getTypeIcon(guitar.guitar_type)" size="20" class="mr-2" />
                  {{ guitar.guitar_type.toUpperCase() }}
                </v-chip>
              </div>
            </div>
          </v-col>

          <v-col cols="12" lg="7">
            <div class="product-info">
              <div class="brand-name text-subtitle-1 text-medium-emphasis mb-1">
                {{ guitar.brand?.name }}
              </div>
              <h1 class="product-title text-h3 font-weight-bold mb-4">
                {{ guitar.model }}
              </h1>

              <div v-if="guitar.price_range" class="price-card mb-6">
                <div class="price-label text-caption text-medium-emphasis">Price Range</div>
                <div class="price-value text-h4 font-weight-bold">
                  {{ guitar.price_range }}
                </div>
              </div>

              <v-tabs v-model="activeTab" color="primary" class="mb-6 product-tabs">
                <v-tab value="specs">
                  <IconifyIcon icon="mdi-cog" class="mr-2" />
                  Specifications
                </v-tab>
                <v-tab value="history">
                  <IconifyIcon icon="mdi-history" class="mr-2" />
                  History
                </v-tab>
                <v-tab v-if="guitar.players?.length" value="players">
                  <IconifyIcon icon="mdi-account-star" class="mr-2" />
                  Famous Players
                </v-tab>
              </v-tabs>

              <v-window v-model="activeTab" class="tab-content">
                <v-window-item value="specs">
                  <v-card variant="outlined" class="specs-card">
                    <v-list density="compact" aria-label="Guitar specifications">
                      <v-list-item
                        v-for="(value, key) in guitar.specifications"
                        :key="key"
                        class="spec-item"
                      >
                        <template #prepend>
                          <IconifyIcon icon="mdi-circle-small" class="mr-2 text-primary" />
                        </template>
                        <v-list-item-title class="spec-key">
                          {{ formatSpecKey(key as string) }}
                        </v-list-item-title>
                        <v-list-item-subtitle class="spec-value">
                          {{ value || 'N/A' }}
                        </v-list-item-subtitle>
                      </v-list-item>
                    </v-list>
                  </v-card>
                </v-window-item>

                <v-window-item value="history">
                  <v-card variant="outlined" class="history-card">
                    <v-card-text v-if="guitar.history" class="text-body-1">
                      {{ guitar.history }}
                    </v-card-text>
                    <v-card-text v-else class="text-center text-medium-emphasis pa-8">
                      <IconifyIcon icon="mdi-history" size="48" class="mb-2" />
                      <p>No history available for this guitar.</p>
                    </v-card-text>
                  </v-card>
                </v-window-item>

                <v-window-item value="players">
                  <v-card variant="outlined" class="players-card">
                    <v-card-text>
                      <div class="players-grid">
                        <v-card
                          v-for="player in guitar.players"
                          :key="player.id"
                          :to="`/players/${player.id}`"
                          variant="outlined"
                          class="player-card"
                        >
                          <v-card-text class="text-center pa-4">
                            <v-avatar size="64" color="primary" class="mb-3">
                              <NuxtImg
                                v-if="player.image_url"
                                :src="player.image_url"
                                :alt="player.name"
                                width="64"
                                height="64"
                                loading="lazy"
                                format="webp"
                                class="player-image"
                              />
                              <IconifyIcon v-else icon="mdi-account" size="32" />
                            </v-avatar>
                            <h4 class="text-subtitle-1 font-weight-bold">{{ player.name }}</h4>
                            <p v-if="player.genre" class="text-caption text-medium-emphasis">
                              {{ player.genre }}
                            </p>
                          </v-card-text>
                        </v-card>
                      </div>
                    </v-card-text>
                  </v-card>
                </v-window-item>
              </v-window>

              <PurchaseLinks
                v-if="guitar.purchase_links"
                :links="guitar.purchase_links"
                class="mt-6"
              />
            </div>
          </v-col>
        </v-row>
      </template>
    </v-container>
  </div>
</template>

<script setup lang="ts">
const route = useRoute();
const { currentGuitar: guitar, loading, error, fetchGuitarById } = useGuitars();

const activeTab = ref('specs');

const breadcrumbs = computed(() => [
  { title: 'Home', to: '/', disabled: false },
  { title: 'Guitars', to: '/guitars', disabled: false },
  { title: guitar.value?.model || 'Loading...', disabled: true },
]);

const getTypeColor = (type: string) => {
  const colors: Record<string, string> = {
    electric: 'red',
    acoustic: 'green',
    bass: 'blue',
  };
  return colors[type] || 'grey';
};

const getTypeIcon = (type: string) => {
  const icons: Record<string, string> = {
    electric: 'mdi-guitar-electric',
    acoustic: 'mdi-guitar-acoustic',
    bass: 'mdi-speak',
  };
  return icons[type] || 'mdi-guitar';
};

const formatSpecKey = (key: string) => {
  return key.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase());
};

useHead({
  title: computed(() =>
    guitar.value ? `${guitar.value.brand?.name} ${guitar.value.model}` : 'Guitar Details',
  ),
});

const id = route.params.id as string;
await fetchGuitarById(id);
</script>

<style scoped>
.detail-page {
  min-height: 100%;
  background: linear-gradient(
    180deg,
    rgba(var(--v-theme-background)) 0%,
    rgba(var(--v-theme-surface)) 100%
  );
}

.loading-state {
  padding: 40px 0;
}

.image-gallery {
  position: sticky;
  top: 80px;
}

.main-image-card {
  overflow: hidden;
  border-radius: 16px;
}

.main-image-wrapper {
  position: relative;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

.main-image {
  width: 100%;
  height: auto;
  display: block;
}

.image-placeholder {
  width: 100%;
  height: 400px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  text-align: center;
  padding: 2rem;
}

.type-badge-wrapper {
  display: flex;
  justify-content: center;
}

.product-info {
  padding-left: 24px;
}

@media (max-width: 1264px) {
  .product-info {
    padding-left: 0;
  }
}

.brand-name {
  text-transform: uppercase;
  letter-spacing: 1px;
  font-size: 0.875rem;
}

.product-title {
  line-height: 1.2;
}

.price-card {
  padding: 20px 24px;
  background: rgba(var(--v-theme-surface));
  border: 2px solid rgba(var(--v-theme-primary), 0.2);
  border-radius: 12px;
}

.price-label {
  margin-bottom: 4px;
}

.price-value {
  color: rgb(var(--v-theme-secondary));
}

.product-tabs {
  border-bottom: 1px solid rgba(var(--v-theme-primary), 0.1);
}

.tab-content {
  min-height: 200px;
}

.specs-card,
.history-card,
.players-card {
  border-radius: 12px;
}

.spec-item {
  border-bottom: 1px solid rgba(var(--v-theme-primary), 0.05);
}

.spec-item:last-child {
  border-bottom: none;
}

.spec-key {
  font-weight: 500;
  color: rgba(255, 255, 255, 0.9);
}

.spec-value {
  color: rgb(var(--v-theme-primary));
  font-weight: 500;
}

.players-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 16px;
}

.player-card {
  transition: all 0.3s ease;
  cursor: pointer;
}

.player-card:hover {
  transform: translateY(-4px);
}

.player-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>

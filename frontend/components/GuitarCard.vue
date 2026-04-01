<template>
  <v-card
    :to="`/guitars/${guitar.id}`"
    class="guitar-card h-100"
    :class="{ 'selected-for-compare': isSelected }"
    hover
  >
    <div class="image-wrapper">
      <NuxtImg
        v-if="guitar.image_url && !guitar.image_url.startsWith('https://via.placeholder')"
        :src="guitar.image_url"
        :alt="`${guitar.brand?.name} ${guitar.model}`"
        width="400"
        height="300"
        loading="lazy"
        format="webp"
        quality="80"
        class="guitar-image"
      />
      <div v-else class="guitar-image-placeholder">
        <IconifyIcon icon="mdi-guitar-electric" size="48" />
        <span class="mt-2">{{ guitar.brand?.name }} {{ guitar.model }}</span>
      </div>

      <div class="image-overlay">
        <v-chip :color="getTypeColor(guitar.guitar_type)" size="small" class="type-badge">
          <IconifyIcon :icon="getTypeIcon(guitar.guitar_type)" size="14" class="mr-1" />
          {{ guitar.guitar_type }}
        </v-chip>

        <v-btn
          icon
          size="small"
          :color="isSelected ? 'success' : 'white'"
          class="compare-btn"
          :aria-label="isSelected ? 'Remove from compare' : 'Add to compare'"
          @click.prevent="toggleCompare"
        >
          <IconifyIcon :icon="isSelected ? 'mdi-check' : 'mdi-plus'" size="18" />
        </v-btn>
      </div>
    </div>

    <v-card-text class="card-content">
      <div class="brand-name text-caption text-medium-emphasis">
        {{ guitar.brand?.name }}
      </div>
      <h3 class="model-name text-subtitle-1 font-weight-bold mb-2">
        {{ guitar.model }}
      </h3>

      <div v-if="guitar.price_range" class="price-section mb-3">
        <IconifyIcon icon="mdi-currency-usd" color="secondary" size="16" class="mr-1" />
        <span class="price-text">{{ extractPriceRange(guitar.price_range) }}</span>
      </div>

      <div v-if="guitar.specifications" class="specs-preview">
        <div v-if="guitar.specifications.body_wood" class="spec-item">
          <IconifyIcon icon="mdi-tree" size="14" class="mr-1 text-medium-emphasis" />
          <span>{{ guitar.specifications.body_wood }}</span>
        </div>
        <div v-if="guitar.specifications.pickup_config" class="spec-item">
          <IconifyIcon icon="mdi-tune" size="14" class="mr-1 text-medium-emphasis" />
          <span>{{ guitar.specifications.pickup_config }}</span>
        </div>
      </div>
    </v-card-text>

    <v-card-actions class="card-actions">
      <v-btn color="primary" variant="tonal" block>
        View Details
        <IconifyIcon icon="mdi-arrow-right" size="18" class="ml-1" />
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

<script setup lang="ts">
import type { Guitar } from '~/types';
import { useComparisonStore } from '~/stores/comparison';

const props = defineProps<{
  guitar: Guitar;
}>();

const comparisonStore = useComparisonStore();
const isSelected = computed(() => comparisonStore.isSelected(props.guitar.id));

const toggleCompare = () => {
  if (isSelected.value) {
    comparisonStore.removeGuitar(props.guitar.id);
  } else {
    comparisonStore.addGuitar(props.guitar);
  }
};

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

const extractPriceRange = (priceRange: string) => {
  const usdMatch = priceRange.match(/\$[\d,]+/);
  return usdMatch ? usdMatch[0].replace(/,/g, ' ') : priceRange;
};
</script>

<style scoped>
.guitar-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.guitar-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3) !important;
}

.guitar-card.selected-for-compare {
  border: 2px solid rgb(var(--v-theme-primary));
}

.image-wrapper {
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

.guitar-image {
  width: 100%;
  height: 220px;
  object-fit: cover;
  transition: transform 0.5s cubic-bezier(0.4, 0, 0.2, 1);
}

.guitar-card:hover .guitar-image {
  transform: scale(1.08);
}

.guitar-image-placeholder {
  width: 100%;
  height: 220px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  font-size: 0.9rem;
  font-weight: 500;
  text-align: center;
  padding: 1rem;
}

.image-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(
    to bottom,
    rgba(0, 0, 0, 0.1) 0%,
    transparent 40%,
    rgba(0, 0, 0, 0.6) 100%
  );
  opacity: 0;
  transition: opacity 0.3s ease;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 12px;
}

.guitar-card:hover .image-overlay {
  opacity: 1;
}

.type-badge {
  backdrop-filter: blur(8px);
  background: rgba(0, 0, 0, 0.5) !important;
  color: white !important;
  text-transform: capitalize;
}

.compare-btn {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  backdrop-filter: blur(8px);
}

.compare-btn:hover {
  transform: scale(1.1);
}

.card-content {
  padding: 16px;
}

.brand-name {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.model-name {
  font-size: 1rem;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.price-section {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: rgba(var(--v-theme-surface), 0.8);
  border-radius: 8px;
  border: 1px solid rgba(var(--v-theme-primary), 0.2);
}

.price-text {
  font-size: 0.9rem;
  font-weight: 600;
  color: rgb(var(--v-theme-secondary));
}

.specs-preview {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.spec-item {
  display: flex;
  align-items: center;
  font-size: 0.8rem;
  color: rgba(255, 255, 255, 0.7);
}

.spec-item span {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-actions {
  padding: 8px 16px 16px;
}

.card-actions :deep(.v-btn) {
  text-transform: none;
  font-weight: 500;
  letter-spacing: 0;
}
</style>

<template>
  <v-card color="surface" class="brand-header">
    <v-row no-gutters>
      <v-col cols="12" md="4" class="d-flex align-center justify-center pa-6">
        <v-avatar size="150" color="primary" class="brand-logo">
          <NuxtImg
            v-if="brand.logo_url"
            :src="brand.logo_url"
            :alt="`${brand.name} logo`"
            width="150"
            height="150"
            sizes="sm:100px md:150px lg:200px"
            loading="lazy"
            format="webp"
            quality="80"
            class="brand-logo-image"
          />
          <IconifyIcon v-else icon="mdi-guitar-electric" size="80" aria-hidden="true" />
        </v-avatar>
      </v-col>

      <v-col cols="12" md="8">
        <v-card-text class="pa-6">
          <h1 class="text-h3 font-weight-bold mb-2">{{ brand.name }}</h1>

          <div class="d-flex align-center gap-4 mb-4">
            <v-chip color="secondary" variant="flat">
              <IconifyIcon icon="mdi-earth" class="mr-1" aria-hidden="true" />
              {{ brand.country }}
            </v-chip>

            <v-chip v-if="brand.founded_year" color="primary" variant="outlined">
              <IconifyIcon icon="mdi-calendar" class="mr-1" aria-hidden="true" />
              Founded {{ brand.founded_year }}
            </v-chip>
          </div>

          <p v-if="brand.description" class="text-body-1 text-medium-emphasis">
            {{ brand.description }}
          </p>

          <v-divider class="my-4" />

          <div class="d-flex gap-2">
            <v-btn color="primary" :to="`/guitars?brand=${brand.id}`">
              <IconifyIcon icon="mdi-guitar-electric" class="mr-2" aria-hidden="true" />
              View Guitars
            </v-btn>
            <v-btn variant="outlined" aria-label="Share this brand" @click="shareBrand">
              <IconifyIcon icon="mdi-share-variant" class="mr-2" aria-hidden="true" />
              Share
            </v-btn>
          </div>
        </v-card-text>
      </v-col>
    </v-row>
  </v-card>
</template>

<script setup lang="ts">
import type { Brand } from '~/types';

defineProps<{
  brand: Brand;
}>();

const shareBrand = async () => {
  if (navigator.share) {
    try {
      await navigator.share({
        title: 'Check out this guitar brand',
        url: window.location.href,
      });
    } catch {
      console.log('Share cancelled');
    }
  }
};
</script>

<style scoped>
.brand-header {
  overflow: hidden;
}

.brand-logo {
  border: 4px solid rgba(var(--v-theme-primary), 0.3);
}

.brand-logo-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.gap-2 {
  gap: 0.5rem;
}

.gap-4 {
  gap: 1rem;
}
</style>

<template>
  <div class="home-page">
    <div class="hero-section">
      <div class="hero-background">
        <div class="hero-gradient" />
        <div class="hero-pattern" />
      </div>

      <v-container class="hero-content">
        <v-row align="center" justify="center">
          <v-col cols="12" md="10" lg="8">
            <div class="hero-text text-center">
              <div class="hero-badge mb-4">
                <IconifyIcon icon="mdi-fire" class="mr-1" />
                Discover Your Perfect Sound
              </div>
              <h1 class="hero-title mb-4">
                <span class="hero-title-line">Find Your</span>
                <span class="hero-title-accent">Dream Guitar</span>
              </h1>
              <p class="hero-subtitle mb-8">
                Explore our curated collection of electric, acoustic, and bass guitars from the
                world's finest manufacturers
              </p>
              <div class="hero-actions">
                <v-btn
                  to="/guitars"
                  size="x-large"
                  color="secondary"
                  class="hero-btn-primary mr-4 mb-2"
                >
                  <IconifyIcon icon="mdi-magnify" class="mr-2" />
                  Explore Catalog
                </v-btn>
                <v-btn
                  to="/brands"
                  size="x-large"
                  variant="outlined"
                  color="white"
                  class="hero-btn-secondary mb-2"
                >
                  <IconifyIcon icon="mdi-factory" class="mr-2" />
                  Browse Brands
                </v-btn>
              </div>
            </div>
          </v-col>
        </v-row>
      </v-container>

      <div class="hero-stats">
        <v-container>
          <v-row justify="center">
            <v-col v-for="stat in stats" :key="stat.label" cols="6" sm="3" md="2">
              <div class="stat-item">
                <div class="stat-value">{{ stat.value }}</div>
                <div class="stat-label">{{ stat.label }}</div>
              </div>
            </v-col>
          </v-row>
        </v-container>
      </div>
    </div>

    <div class="featured-section">
      <v-container>
        <div class="section-header mb-8">
          <div class="section-title-group">
            <div class="section-badge">
              <IconifyIcon icon="mdi-star" size="16" />
              Featured
            </div>
            <h2 class="section-title">Featured Guitars</h2>
            <p class="section-subtitle">Hand-picked selections from our collection</p>
          </div>
          <v-btn to="/guitars" variant="text" color="primary" class="view-all-btn">
            View All
            <IconifyIcon icon="mdi-arrow-right" class="ml-1" />
          </v-btn>
        </div>

        <div v-if="loading" class="featured-grid">
          <v-skeleton-loader v-for="n in 4" :key="n" type="card" class="skeleton-card" />
        </div>

        <div v-else class="featured-grid">
          <GuitarCard v-for="guitar in featuredGuitars" :key="guitar.id" :guitar="guitar" />
        </div>
      </v-container>
    </div>

    <div class="categories-section">
      <v-container>
        <div class="section-header mb-8">
          <div class="section-title-group">
            <div class="section-badge">
              <IconifyIcon icon="mdi-shape" size="16" />
              Categories
            </div>
            <h2 class="section-title">Shop by Type</h2>
            <p class="section-subtitle">Find the perfect guitar for your style</p>
          </div>
        </div>

        <v-row>
          <v-col v-for="category in categories" :key="category.type" cols="12" sm="4">
            <v-card
              :to="`/guitars?type=${category.type}`"
              class="category-card"
              :color="category.color"
              variant="tonal"
            >
              <v-card-text class="text-center pa-8">
                <v-avatar size="80" :color="category.color" class="mb-4">
                  <IconifyIcon :icon="category.icon" size="40" />
                </v-avatar>
                <h3 class="text-h5 font-weight-bold mb-2">{{ category.name }}</h3>
                <p class="text-body-2 text-medium-emphasis">{{ category.description }}</p>
                <v-btn variant="outlined" size="small" :color="category.color" class="mt-4">
                  Explore
                  <IconifyIcon icon="mdi-arrow-right" size="16" class="ml-1" />
                </v-btn>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
      </v-container>
    </div>

    <div class="brands-section">
      <v-container>
        <div class="section-header mb-8">
          <div class="section-title-group">
            <div class="section-badge">
              <IconifyIcon icon="mdi-factory" size="16" />
              Brands
            </div>
            <h2 class="section-title">Popular Brands</h2>
            <p class="section-subtitle">Discover world-renowned guitar manufacturers</p>
          </div>
          <v-btn to="/brands" variant="text" color="primary" class="view-all-btn">
            All Brands
            <IconifyIcon icon="mdi-arrow-right" class="ml-1" />
          </v-btn>
        </div>

        <div v-if="brandsLoading" class="brands-grid">
          <v-skeleton-loader v-for="n in 6" :key="n" type="card" class="brand-skeleton" />
        </div>

        <div v-else class="brands-grid">
          <v-card
            v-for="brand in featuredBrands"
            :key="brand.id"
            :to="`/brands/${brand.id}`"
            class="brand-card"
          >
            <v-card-text class="text-center pa-6">
              <v-avatar size="72" color="primary" class="brand-avatar mb-4">
                <NuxtImg
                  v-if="brand.logo_url"
                  :src="brand.logo_url"
                  :alt="`${brand.name} logo`"
                  width="72"
                  height="72"
                  loading="lazy"
                  format="webp"
                  class="brand-logo"
                />
                <IconifyIcon v-else icon="mdi-guitar-electric" size="32" />
              </v-avatar>
              <h3 class="text-subtitle-1 font-weight-bold mb-1">{{ brand.name }}</h3>
              <div class="text-caption text-medium-emphasis">{{ brand.country }}</div>
            </v-card-text>
          </v-card>
        </div>
      </v-container>
    </div>

    <div class="cta-section">
      <v-container>
        <v-card color="primary" class="cta-card">
          <v-card-text class="pa-12 text-center">
            <IconifyIcon icon="mdi-guitar-acoustic" size="64" color="white" class="mb-4" />
            <h2 class="text-h4 font-weight-bold text-white mb-4">
              Ready to Find Your Perfect Guitar?
            </h2>
            <p class="text-body-1 text-white-darken-2 mb-6 mx-auto" style="max-width: 600px">
              Browse our extensive catalog with detailed specifications, famous player associations,
              and purchase links from trusted retailers.
            </p>
            <div class="d-flex justify-center flex-wrap">
              <v-btn to="/guitars" size="large" color="secondary" class="mr-4 mb-2">
                <IconifyIcon icon="mdi-magnify" class="mr-2" />
                Browse Guitars
              </v-btn>
              <v-btn to="/compare" size="large" variant="outlined" color="white" class="mb-2">
                <IconifyIcon icon="mdi-compare-horizontal" class="mr-2" />
                Compare Guitars
              </v-btn>
            </div>
          </v-card-text>
        </v-card>
      </v-container>
    </div>
  </div>
</template>

<script setup lang="ts">
const { guitars, loading: guitarsLoading, fetchGuitars } = useGuitars();
const { brands, loading: brandsLoading, fetchBrands } = useBrands();

const loading = computed(() => guitarsLoading.value);

const featuredGuitars = computed(() => guitars.value.slice(0, 4));
const featuredBrands = computed(() => brands.value.slice(0, 6));

const stats = [
  { value: '100+', label: 'Guitars' },
  { value: '12+', label: 'Brands' },
  { value: '3', label: 'Types' },
  { value: '50+', label: 'Players' },
];

const categories = [
  {
    type: 'electric',
    name: 'Electric Guitars',
    description: 'Rock, metal, blues, jazz - find your sound',
    icon: 'mdi-guitar-electric',
    color: 'red',
  },
  {
    type: 'acoustic',
    name: 'Acoustic Guitars',
    description: 'Folk, fingerstyle, singer-songwriter',
    icon: 'mdi-guitar-acoustic',
    color: 'green',
  },
  {
    type: 'bass',
    name: 'Bass Guitars',
    description: 'Low end that drives the rhythm',
    icon: 'mdi-speak',
    color: 'blue',
  },
];

useHead({
  title: 'Guitar Stock - Your Guitar Catalog',
});

await Promise.all([fetchGuitars({ limit: 4 }), fetchBrands()]);
</script>

<style scoped>
.home-page {
  min-height: 100%;
}

.hero-section {
  position: relative;
  padding: 80px 0 0;
  overflow: hidden;
}

.hero-background {
  position: absolute;
  inset: 0;
  z-index: 0;
}

.hero-gradient {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    rgba(147, 51, 234, 0.9) 0%,
    rgba(79, 70, 229, 0.9) 50%,
    rgba(147, 51, 234, 0.9) 100%
  );
}

.hero-pattern {
  position: absolute;
  inset: 0;
  background-image: radial-gradient(rgba(255, 255, 255, 0.1) 1px, transparent 1px);
  background-size: 30px 30px;
}

.hero-content {
  position: relative;
  z-index: 1;
}

.hero-text {
  color: white;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(10px);
  border-radius: 50px;
  font-size: 0.875rem;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.95);
}

.hero-title {
  font-size: clamp(2.5rem, 6vw, 4rem);
  font-weight: 800;
  line-height: 1.1;
}

.hero-title-line {
  display: block;
  opacity: 0.9;
}

.hero-title-accent {
  display: block;
  color: #ffb300;
  text-shadow: 0 4px 20px rgba(255, 179, 0, 0.3);
}

.hero-subtitle {
  font-size: 1.125rem;
  line-height: 1.6;
  opacity: 0.9;
  max-width: 600px;
  margin-left: auto;
  margin-right: auto;
}

.hero-actions {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
}

.hero-btn-primary {
  font-weight: 600;
  text-transform: none;
  letter-spacing: 0;
  box-shadow: 0 4px 20px rgba(255, 179, 0, 0.4);
}

.hero-btn-primary:hover {
  box-shadow: 0 6px 30px rgba(255, 179, 0, 0.5);
  transform: translateY(-2px);
}

.hero-btn-secondary {
  font-weight: 600;
  text-transform: none;
  letter-spacing: 0;
}

.hero-btn-secondary:hover {
  background: rgba(255, 255, 255, 0.1);
  transform: translateY(-2px);
}

.hero-stats {
  position: relative;
  z-index: 1;
  margin-top: 60px;
  padding: 24px 0;
  background: rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(10px);
}

.stat-item {
  text-align: center;
  color: white;
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  line-height: 1;
  margin-bottom: 4px;
}

.stat-label {
  font-size: 0.875rem;
  opacity: 0.8;
}

.featured-section,
.categories-section,
.brands-section,
.cta-section {
  padding: 80px 0;
}

.featured-section {
  background: linear-gradient(
    to bottom,
    rgba(var(--v-theme-background)) 0%,
    rgba(var(--v-theme-surface)) 100%
  );
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  flex-wrap: wrap;
  gap: 16px;
}

.section-title-group {
  flex: 1;
  min-width: 200px;
}

.section-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: rgba(var(--v-theme-primary), 0.1);
  color: rgb(var(--v-theme-primary));
  border-radius: 50px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.section-title {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 8px;
}

.section-subtitle {
  color: rgba(255, 255, 255, 0.6);
  font-size: 1rem;
}

.view-all-btn {
  text-transform: none;
  font-weight: 500;
}

.featured-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

@media (max-width: 1200px) {
  .featured-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 600px) {
  .featured-grid {
    grid-template-columns: 1fr;
  }
}

.skeleton-card {
  height: 400px;
}

.category-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
}

.category-card:hover {
  transform: translateY(-8px);
}

.brands-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 16px;
}

@media (max-width: 1200px) {
  .brands-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 600px) {
  .brands-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

.brand-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
}

.brand-card:hover {
  transform: translateY(-4px);
}

.brand-avatar {
  border: 3px solid rgba(var(--v-theme-primary), 0.3);
  transition: border-color 0.3s ease;
}

.brand-card:hover .brand-avatar {
  border-color: rgba(var(--v-theme-primary), 0.6);
}

.brand-logo {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cta-card {
  background: linear-gradient(135deg, rgba(147, 51, 234, 0.95), rgba(79, 70, 229, 0.95)) !important;
  border: none;
}
</style>

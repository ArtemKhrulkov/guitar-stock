<template>
  <v-app>
    <v-app-bar
      :elevation="scrolled ? 4 : 0"
      :color="scrolled ? 'surface' : 'transparent'"
      class="app-navbar"
      :class="{ 'navbar-scrolled': scrolled }"
    >
      <v-container class="d-flex align-center py-0" style="max-width: 1400px">
        <v-app-bar-nav-icon
          class="d-md-none"
          aria-label="Open navigation menu"
          @click="drawer = !drawer"
        />

        <NuxtLink to="/" class="navbar-brand">
          <div class="brand-icon">
            <IconifyIcon icon="mdi-guitar-acoustic" size="28" />
          </div>
          <span class="brand-text">Guitar Stock</span>
        </NuxtLink>

        <v-spacer />

        <div class="nav-links d-none d-md-flex align-center ga-1">
          <NuxtLink to="/" class="nav-link" :class="{ active: route.path === '/' }">
            <IconifyIcon icon="mdi-home" size="18" class="mr-1" />
            Home
          </NuxtLink>
          <NuxtLink
            to="/guitars"
            class="nav-link"
            :class="{ active: route.path.startsWith('/guitars') }"
          >
            <IconifyIcon icon="mdi-guitar-electric" size="18" class="mr-1" />
            Guitars
          </NuxtLink>
          <NuxtLink
            to="/brands"
            class="nav-link"
            :class="{ active: route.path.startsWith('/brands') }"
          >
            <IconifyIcon icon="mdi-factory" size="18" class="mr-1" />
            Brands
          </NuxtLink>
          <NuxtLink to="/compare" class="nav-link" :class="{ active: route.path === '/compare' }">
            <IconifyIcon icon="mdi-compare-horizontal" size="18" class="mr-1" />
            Compare
          </NuxtLink>
        </div>

        <v-spacer />

        <div class="nav-actions d-flex align-center ga-2">
          <v-menu
            v-model="searchMenuOpen"
            :close-on-content-click="false"
            location="bottom end"
            offset="8"
          >
            <template #activator="{ props }">
              <v-btn icon variant="text" v-bind="props" aria-label="Search" class="action-btn">
                <IconifyIcon icon="mdi-magnify" size="22" />
              </v-btn>
            </template>
            <v-card min-width="320" class="search-card" elevation="8">
              <v-card-text class="pa-3">
                <v-text-field
                  v-model="searchQuery"
                  placeholder="Search guitars, players..."
                  prepend-inner-icon="mdi-magnify"
                  variant="solo"
                  hide-details
                  density="compact"
                  autofocus
                  class="search-input"
                  @keyup.enter="performSearch"
                />
              </v-card-text>
            </v-card>
          </v-menu>
        </div>
      </v-container>
    </v-app-bar>

    <v-navigation-drawer v-model="drawer" temporary class="mobile-drawer">
      <v-list nav>
        <v-list-item title="Menu" subtitle="Navigation" class="drawer-header" />
        <v-divider class="mb-2" />
        <v-list-item to="/" prepend-icon="mdi-home" title="Home" @click="drawer = false" />
        <v-list-item
          to="/guitars"
          prepend-icon="mdi-guitar-electric"
          title="Guitars"
          @click="drawer = false"
        />
        <v-list-item
          to="/brands"
          prepend-icon="mdi-factory"
          title="Brands"
          @click="drawer = false"
        />
        <v-list-item
          to="/compare"
          prepend-icon="mdi-compare-horizontal"
          title="Compare"
          @click="drawer = false"
        />
      </v-list>
    </v-navigation-drawer>

    <v-main id="main-content" class="main-content">
      <slot />
    </v-main>

    <ComparisonBar />

    <v-footer class="site-footer py-6">
      <v-container style="max-width: 1400px">
        <v-row justify="center" align="center">
          <v-col cols="12" md="4" class="text-center text-md-left mb-4 mb-md-0">
            <div class="footer-brand">
              <IconifyIcon icon="mdi-guitar-acoustic" size="24" class="mr-2" />
              <span class="text-h6">Guitar Stock</span>
            </div>
            <p class="text-caption text-medium-emphasis mt-2">
              Your ultimate guitar catalog with detailed specifications, famous player associations,
              and purchase links.
            </p>
          </v-col>

          <v-col cols="12" md="4" class="text-center mb-4 mb-md-0">
            <div class="footer-links">
              <NuxtLink to="/guitars" class="footer-link">Guitars</NuxtLink>
              <NuxtLink to="/brands" class="footer-link">Brands</NuxtLink>
              <NuxtLink to="/compare" class="footer-link">Compare</NuxtLink>
            </div>
          </v-col>

          <v-col cols="12" md="4" class="text-center text-md-right">
            <p class="text-caption text-medium-emphasis">Built with Vue 3, Nuxt 3, and Vuetify</p>
            <p class="text-caption text-disabled mt-1">
              {{ currentYear }}
            </p>
          </v-col>
        </v-row>
      </v-container>
    </v-footer>
  </v-app>
</template>

<script setup lang="ts">
const route = useRoute();
const router = useRouter();

const drawer = ref(false);
const searchQuery = ref('');
const searchMenuOpen = ref(false);
const scrolled = ref(false);

const performSearch = () => {
  if (!searchQuery.value.trim()) return;
  router.push({ path: '/guitars', query: { search: searchQuery.value } });
  searchMenuOpen.value = false;
  searchQuery.value = '';
};

const currentYear = new Date().getFullYear();

onMounted(() => {
  window.addEventListener('scroll', () => {
    scrolled.value = window.scrollY > 20;
  });
});

onUnmounted(() => {
  window.removeEventListener('scroll', () => {
    scrolled.value = false;
  });
});
</script>

<style scoped>
.app-navbar {
  transition: all 0.3s ease;
  border-bottom: 1px solid transparent;
}

.navbar-scrolled {
  border-bottom-color: rgba(var(--v-theme-primary), 0.1);
  backdrop-filter: blur(10px);
}

.navbar-brand {
  display: flex;
  align-items: center;
  text-decoration: none;
  color: inherit;
  gap: 12px;
}

.brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, rgba(var(--v-theme-primary)), rgba(var(--v-theme-secondary)));
  border-radius: 12px;
  color: white;
}

.brand-text {
  font-size: 1.25rem;
  font-weight: 700;
}

.nav-links {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
}

.nav-link {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  border-radius: 8px;
  text-decoration: none;
  color: rgba(255, 255, 255, 0.7);
  font-weight: 500;
  font-size: 0.9rem;
  transition: all 0.2s ease;
}

.nav-link:hover {
  color: white;
  background: rgba(var(--v-theme-primary), 0.1);
}

.nav-link.active {
  color: rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.15);
}

.action-btn {
  transition: all 0.2s ease;
}

.action-btn:hover {
  background: rgba(var(--v-theme-primary), 0.1);
}

.search-card {
  border-radius: 12px;
}

.search-input :deep(.v-field) {
  border-radius: 8px;
}

.mobile-drawer {
  background: rgba(var(--v-theme-surface)) !important;
}

.drawer-header {
  padding: 16px;
}

.main-content {
  min-height: calc(100vh - 200px);
}

.site-footer {
  background: rgba(var(--v-theme-surface));
  border-top: 1px solid rgba(var(--v-theme-primary), 0.1);
}

.footer-brand {
  display: flex;
  align-items: center;
  justify-content: center;
  justify-content: flex-start;
  color: inherit;
}

.footer-links {
  display: flex;
  gap: 24px;
  justify-content: center;
  flex-wrap: wrap;
}

.footer-link {
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  font-size: 0.9rem;
  transition: color 0.2s ease;
}

.footer-link:hover {
  color: rgb(var(--v-theme-primary));
}

@media (max-width: 960px) {
  .nav-links {
    position: static;
    transform: none;
  }
}
</style>

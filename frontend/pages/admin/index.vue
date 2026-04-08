<template>
  <div class="min-h-screen bg-gray-900">
    <header class="bg-gray-800 border-b border-gray-700">
      <div class="max-w-7xl mx-auto px-4 py-4 flex justify-between items-center">
        <h1 class="text-xl font-bold text-white">Admin Dashboard</h1>
        <v-btn color="error" variant="outlined" size="small" @click="handleLogout"> Logout </v-btn>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-4 py-8">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <v-card class="pa-6">
          <div class="flex items-center gap-4">
            <IconifyIcon icon="mdi:link-variant" class="text-4xl text-primary" />
            <div>
              <div class="text-3xl font-bold">
                {{ totalLinks }}
              </div>
              <div class="text-gray-400">Total Purchase Links</div>
            </div>
          </div>
        </v-card>

        <v-card class="pa-6">
          <div class="flex items-center gap-4">
            <IconifyIcon icon="mdi:guitar-acoustic" class="text-4xl text-secondary" />
            <div>
              <div class="text-3xl font-bold">
                {{ totalGuitars }}
              </div>
              <div class="text-gray-400">Total Guitars</div>
            </div>
          </div>
        </v-card>
      </div>

      <h2 class="text-lg font-semibold text-white mb-4">Content Management</h2>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <v-card
          class="pa-6 cursor-pointer hover:border-primary transition-colors"
          @click="navigateTo('/admin/links')"
        >
          <div class="flex items-center gap-4">
            <IconifyIcon icon="mdi:link-plus" class="text-3xl text-primary" />
            <div>
              <div class="font-semibold">Manage Links</div>
              <div class="text-sm text-gray-400">Add and remove purchase links</div>
            </div>
          </div>
        </v-card>

        <v-card
          class="pa-6 cursor-pointer hover:border-primary transition-colors"
          @click="navigateTo('/admin/guitars')"
        >
          <div class="flex items-center gap-4">
            <IconifyIcon icon="mdi:music-box-multiple" class="text-3xl text-secondary" />
            <div>
              <div class="font-semibold">Browse Guitars</div>
              <div class="text-sm text-gray-400">View and manage guitars</div>
            </div>
          </div>
        </v-card>

        <v-card
          class="pa-6 cursor-pointer hover:border-primary transition-colors"
          @click="navigateTo('/admin/brands')"
        >
          <div class="flex items-center gap-4">
            <IconifyIcon icon="mdi:factory" class="text-3xl text-warning" />
            <div>
              <div class="font-semibold">Manage Brands</div>
              <div class="text-sm text-gray-400">CRUD for guitar brands</div>
            </div>
          </div>
        </v-card>

        <v-card
          class="pa-6 cursor-pointer hover:border-primary transition-colors"
          @click="navigateTo('/admin/players')"
        >
          <div class="flex items-center gap-4">
            <IconifyIcon icon="mdi:account-music" class="text-3xl text-success" />
            <div>
              <div class="font-semibold">Manage Players</div>
              <div class="text-sm text-gray-400">CRUD for famous players</div>
            </div>
          </div>
        </v-card>
      </div>

      <h2 class="text-lg font-semibold text-white mb-4">Scraping Controls</h2>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <v-card class="pa-6">
          <div class="flex items-center gap-4 mb-4">
            <IconifyIcon icon="mdi:spider" class="text-3xl text-purple" />
            <div>
              <div class="font-semibold">Scrape All Guitars</div>
              <div class="text-sm text-gray-400">Update prices for all guitars</div>
            </div>
          </div>
          <v-btn
            color="purple"
            :loading="scrapeAllLoading"
            :disabled="scrapeAllLoading"
            block
            @click="scrapeAll"
          >
            <IconifyIcon icon="mdi:play" class="mr-2" />
            Start Scrape
          </v-btn>
        </v-card>

        <v-card class="pa-6">
          <div class="flex items-center gap-4 mb-4">
            <IconifyIcon icon="mdi:sync" class="text-3xl text-info" />
            <div>
              <div class="font-semibold">Sync Price Ranges</div>
              <div class="text-sm text-gray-400">Update guitar price ranges</div>
            </div>
          </div>
          <v-btn
            color="info"
            :loading="syncLoading"
            :disabled="syncLoading"
            block
            @click="syncPrices"
          >
            <IconifyIcon icon="mdi:refresh" class="mr-2" />
            Sync Now
          </v-btn>
        </v-card>

        <v-card class="pa-6">
          <div class="flex items-center gap-4 mb-4">
            <IconifyIcon icon="mdi:image-multiple" class="text-3xl text-pink" />
            <div>
              <div class="font-semibold">Scrape Images</div>
              <div class="text-sm text-gray-400">Run image scraper CLI</div>
            </div>
          </div>
          <v-btn
            color="pink"
            variant="outlined"
            disabled
            block
            title="Run manually: go run ./backend/cmd/scraper/main.go --all"
          >
            <IconifyIcon icon="mdi:information" class="mr-2" />
            CLI Only
          </v-btn>
        </v-card>
      </div>

      <div v-if="scrapeMessage" class="mt-4">
        <v-alert :type="scrapeError ? 'error' : 'success'" variant="tonal">
          {{ scrapeMessage }}
        </v-alert>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  middleware: ['auth'],
  layout: 'admin',
});

const { logout } = useAuth();
const config = useRuntimeConfig();
const API_BASE = config.public.apiUrl;

const totalLinks = ref(0);
const totalGuitars = ref(0);

const scrapeAllLoading = ref(false);
const syncLoading = ref(false);
const scrapeMessage = ref('');
const scrapeError = ref(false);

const fetchStats = async () => {
  try {
    const guitarsRes = await $fetch<{ guitars: unknown[]; total: number }>(
      `${API_BASE}/guitars?limit=1`,
      {
        credentials: 'include',
      },
    );
    totalGuitars.value = guitarsRes.total;
  } catch {
    console.error('Failed to fetch stats');
  }
};

const fetchLinks = async () => {
  try {
    const linksRes = await $fetch<{ links: unknown[]; total: number }>(
      `${API_BASE}/admin/links?limit=1`,
      {
        credentials: 'include',
      },
    );
    totalLinks.value = linksRes.total;
  } catch {
    console.error('Failed to fetch stats');
  }
};

const scrapeAll = async () => {
  scrapeAllLoading.value = true;
  scrapeMessage.value = '';
  scrapeError.value = false;

  try {
    await $fetch(`${API_BASE}/admin/scrape/all`, {
      method: 'POST',
      credentials: 'include',
    });
    scrapeMessage.value = 'Scraping started successfully. This may take a while.';
  } catch (err: unknown) {
    scrapeError.value = true;
    const e = err as { data?: { error?: string } };
    scrapeMessage.value = e.data?.error || 'Failed to start scraping';
  } finally {
    scrapeAllLoading.value = false;
  }
};

const syncPrices = async () => {
  syncLoading.value = true;
  scrapeMessage.value = '';
  scrapeError.value = false;

  try {
    await $fetch(`${API_BASE}/admin/scrape/sync-price-ranges`, {
      method: 'POST',
      credentials: 'include',
    });
    scrapeMessage.value = 'Price ranges synced successfully.';
    fetchLinks();
  } catch (err: unknown) {
    scrapeError.value = true;
    const e = err as { data?: { error?: string } };
    scrapeMessage.value = e.data?.error || 'Failed to sync prices';
  } finally {
    syncLoading.value = false;
  }
};

const handleLogout = async () => {
  await logout();
  navigateTo('/login');
};

onMounted(() => {
  Promise.all([fetchStats(), fetchLinks()]);
});
</script>

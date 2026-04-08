<template>
  <div class="profile-page">
    <v-container>
      <div class="profile-header mb-6">
        <div class="d-flex align-center justify-space-between flex-wrap">
          <div>
            <h1 class="text-h4 font-weight-bold">My Profile</h1>
            <p class="text-body-1 text-medium-emphasis">{{ user?.email }}</p>
          </div>
          <v-btn color="error" variant="outlined" @click="handleLogout">
            <IconifyIcon icon="mdi-logout" class="mr-2" />
            Logout
          </v-btn>
        </div>
      </div>

      <v-tabs v-model="activeTab" color="primary" class="mb-6">
        <v-tab value="wishlist">
          <IconifyIcon icon="mdi-heart" class="mr-2" />
          My Wishlist
          <v-badge v-if="wishlistCount > 0" :content="wishlistCount" color="error" inline class="ml-2" />
        </v-tab>
        <v-tab value="settings">
          <IconifyIcon icon="mdi-cog" class="mr-2" />
          Settings
        </v-tab>
      </v-tabs>

      <v-window v-model="activeTab">
        <v-window-item value="wishlist">
          <div v-if="wishlistLoading" class="text-center py-12">
            <v-progress-circular indeterminate color="primary" size="48" />
          </div>

          <div v-else-if="wishlistGuitars.length === 0" class="text-center py-12">
            <IconifyIcon icon="mdi-heart-outline" size="64" class="text-medium-emphasis mb-4" />
            <h2 class="text-h6 mb-2">Your wishlist is empty</h2>
            <p class="text-body-2 text-medium-emphasis mb-4">
              Browse our guitar catalog and add your favorites!
            </p>
            <v-btn color="primary" to="/guitars">
              Browse Guitars
            </v-btn>
          </div>

          <div v-else>
            <v-row>
              <v-col
                v-for="guitar in wishlistGuitars"
                :key="guitar.id"
                cols="12"
                sm="6"
                md="4"
                lg="3"
              >
                <v-card class="h-100">
                  <div class="image-wrapper">
                    <NuxtImg
                      v-if="guitar.image_url && !guitar.image_url.startsWith('https://via.placeholder')"
                      :src="guitar.image_url"
                      :alt="`${guitar.brand?.name} ${guitar.model}`"
                      width="400"
                      height="250"
                      loading="lazy"
                      format="webp"
                      quality="80"
                      class="guitar-image"
                    />
                    <div v-else class="guitar-image-placeholder">
                      <IconifyIcon icon="mdi-guitar-electric" size="40" />
                    </div>
                    <v-btn
                      icon
                      size="small"
                      color="error"
                      class="remove-btn"
                      @click="removeFromWishlist(guitar.id)"
                    >
                      <IconifyIcon icon="mdi-close" size="18" />
                    </v-btn>
                  </div>
                  <v-card-text>
                    <div class="text-caption text-medium-emphasis">
                      {{ guitar.brand?.name }}
                    </div>
                    <h3 class="text-subtitle-1 font-weight-bold">
                      {{ guitar.model }}
                    </h3>
                    <div v-if="guitar.price_range" class="mt-2">
                      <IconifyIcon icon="mdi-currency-usd" size="14" class="mr-1" />
                      <span class="text-body-2">{{ extractPrice(guitar.price_range) }}</span>
                    </div>
                  </v-card-text>
                  <v-card-actions>
                    <v-btn color="primary" variant="tonal" :to="`/guitars/${guitar.id}`" block>
                      View Details
                    </v-btn>
                  </v-card-actions>
                </v-card>
              </v-col>
            </v-row>
          </div>
        </v-window-item>

        <v-window-item value="settings">
          <v-card class="pa-6">
            <h2 class="text-h6 mb-4">Account Settings</h2>

            <v-list>
              <v-list-item>
                <template #prepend>
                  <IconifyIcon icon="mdi-email" class="mr-2" />
                </template>
                <v-list-item-title>Email</v-list-item-title>
                <v-list-item-subtitle>{{ user?.email }}</v-list-item-subtitle>
              </v-list-item>

              <v-list-item>
                <template #prepend>
                  <IconifyIcon icon="mdi-account" class="mr-2" />
                </template>
                <v-list-item-title>Role</v-list-item-title>
                <v-list-item-subtitle class="text-capitalize">{{ user?.role }}</v-list-item-subtitle>
              </v-list-item>

              <v-list-item>
                <template #prepend>
                  <IconifyIcon icon="mdi-check-circle" class="mr-2" />
                </template>
                <v-list-item-title>Email Verified</v-list-item-title>
                <v-list-item-subtitle>
                  <v-chip
                    :color="user?.email_verified ? 'success' : 'warning'"
                    size="small"
                  >
                    {{ user?.email_verified ? 'Verified' : 'Pending' }}
                  </v-chip>
                </v-list-item-subtitle>
              </v-list-item>
            </v-list>
          </v-card>
        </v-window-item>
      </v-window>
    </v-container>
  </div>
</template>

<script setup lang="ts">
import type { Guitar } from '~/types';

definePageMeta({
  layout: 'default',
  middleware: 'auth',
});

const { user, logout } = useAuth();
const wishlistStore = useWishlist();

const activeTab = ref('wishlist');
const wishlistGuitars = ref<Guitar[]>([]);
const wishlistLoading = ref(false);

const wishlistCount = computed(() => wishlistStore.guitarIds.value.length);

onMounted(async () => {
  wishlistLoading.value = true;
  try {
    await wishlistStore.fetchWishlist();
    await fetchWishlistGuitars();
  } finally {
    wishlistLoading.value = false;
  }
});

const fetchWishlistGuitars = async () => {
  const ids = wishlistStore.guitarIds.value;
  if (ids.length === 0) {
    wishlistGuitars.value = [];
    return;
  }

  try {
    const response = await $fetch<{ guitars: Guitar[] }>('/api/guitars', {
      params: {
        ids: ids.join(','),
      },
    });
    wishlistGuitars.value = response.guitars || [];
  } catch (error) {
    console.error('Failed to fetch wishlist guitars:', error);
    wishlistGuitars.value = [];
  }
};

const removeFromWishlist = async (guitarId: string) => {
  try {
    await wishlistStore.removeFromWishlist(guitarId);
    wishlistGuitars.value = wishlistGuitars.value.filter(g => g.id !== guitarId);
  } catch (error) {
    console.error('Failed to remove from wishlist:', error);
  }
};

const handleLogout = async () => {
  await logout();
  navigateTo('/');
};

const extractPrice = (priceRange: string) => {
  const usdMatch = priceRange.match(/\$[\d,]+/);
  return usdMatch ? usdMatch[0].replace(/,/g, ' ') : priceRange;
};
</script>

<style scoped>
.profile-page {
  min-height: calc(100vh - 200px);
  padding-top: 2rem;
}

.image-wrapper {
  position: relative;
  overflow: hidden;
}

.guitar-image {
  width: 100%;
  height: 200px;
  object-fit: cover;
}

.guitar-image-placeholder {
  width: 100%;
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.remove-btn {
  position: absolute;
  top: 8px;
  right: 8px;
}
</style>

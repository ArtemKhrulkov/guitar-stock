<template>
    <v-container fluid class="pa-6">
        <!-- Loading State -->
        <v-row v-if="loading">
            <v-col cols="12">
                <v-skeleton-loader type="article"></v-skeleton-loader>
            </v-col>
        </v-row>

        <!-- Error State -->
        <v-card
            v-else-if="error"
            color="error"
            class="pa-6 text-center"
            role="alert"
        >
            <IconifyIcon
                icon="mdi-alert-circle"
                size="48"
                class="mb-4"
                aria-hidden="true"
            ></IconifyIcon>
            <h2 class="text-h5 mb-2">Error Loading Guitar</h2>
            <p class="mb-4">{{ error }}</p>
            <v-btn to="/guitars" color="white" variant="flat">
                Back to Catalog
            </v-btn>
        </v-card>

        <!-- Guitar Detail -->
        <template v-else-if="guitar">
            <!-- Breadcrumbs -->
            <v-breadcrumbs
                :items="breadcrumbs"
                class="px-0 mb-4"
                aria-label="Breadcrumb navigation"
            >
                <template v-slot:divider>
                    <IconifyIcon
                        icon="mdi-chevron-right"
                        class="align-middle"
                        aria-hidden="true"
                    ></IconifyIcon>
                </template>
            </v-breadcrumbs>

            <v-row>
                <!-- Image Section -->
                <v-col cols="12" md="5">
                    <v-card elevation="4" class="overflow-hidden">
                        <NuxtImg
                            v-if="guitar.image_url && !guitar.image_url.startsWith('https://via.placeholder')"
                            :src="guitar.image_url"
                            :alt="`${guitar.brand?.name} ${guitar.model}`"
                            width="600"
                            height="500"
                            loading="eager"
                            format="webp"
                            quality="80"
                            class="guitar-detail-image"
                        />
                        <div v-else class="guitar-detail-image-placeholder">
                            <span>{{ guitar.brand?.name }} {{ guitar.model }}</span>
                        </div>
                    </v-card>

                    <!-- Type Badge -->
                    <div class="mt-4 text-center">
                        <v-chip
                            :color="getTypeColor(guitar.guitar_type)"
                            size="large"
                            class="px-6"
                        >
                            <IconifyIcon
                                :icon="getTypeIcon(guitar.guitar_type)"
                                class="mr-2"
                                aria-hidden="true"
                            ></IconifyIcon>
                            {{ guitar.guitar_type.toUpperCase() }}
                        </v-chip>
                    </div>
                </v-col>

                <!-- Details Section -->
                <v-col cols="12" md="7">
                    <!-- Brand & Model -->
                    <h1 class="text-h3 font-weight-bold mb-2">
                        {{ guitar.brand?.name }}
                    </h1>
                    <h2 class="text-h4 text-medium-emphasis mb-4">
                        {{ guitar.model }}
                    </h2>

                    <!-- Price Range -->
                    <v-card color="surface" class="pa-4 mb-6">
                        <div class="d-flex align-center">
                            <IconifyIcon
                                icon="mdi-currency-usd"
                                color="secondary"
                                class="mr-3"
                                aria-hidden="true"
                            ></IconifyIcon>
                            <div>
                                <div class="text-caption text-medium-emphasis">
                                    Price Range
                                </div>
                                <div class="text-h6 font-weight-bold">
                                    {{
                                        guitar.price_range ||
                                        "Price not available"
                                    }}
                                </div>
                            </div>
                        </div>
                    </v-card>

                    <!-- Specifications -->
                    <v-expansion-panels variant="accordion" class="mb-6">
                        <v-expansion-panel title="Specifications">
                            <v-expansion-panel-text>
                                <v-list
                                    density="compact"
                                    aria-label="Guitar specifications"
                                >
                                    <v-list-item
                                        v-for="(
                                            value, key
                                        ) in guitar.specifications"
                                        :key="key"
                                        :title="formatSpecKey(key)"
                                        :subtitle="value?.toString() || 'N/A'"
                                    >
                                        <template v-slot:prepend>
                                            <IconifyIcon
                                                icon="mdi-cog"
                                                aria-hidden="true"
                                                class="mr-2"
                                            ></IconifyIcon>
                                        </template>
                                    </v-list-item>
                                </v-list>
                            </v-expansion-panel-text>
                        </v-expansion-panel>
                    </v-expansion-panels>

                    <!-- History -->
                    <v-card v-if="guitar.history" class="mb-6">
                        <v-card-title class="d-flex align-center">
                            <IconifyIcon
                                icon="mdi-history"
                                color="primary"
                                class="mr-2"
                                aria-hidden="true"
                            ></IconifyIcon>
                            History
                        </v-card-title>
                        <v-card-text class="text-body-1">
                            {{ guitar.history }}
                        </v-card-text>
                    </v-card>

                    <!-- Famous Players -->
                    <v-card
                        v-if="guitar.players && guitar.players.length > 0"
                        class="mb-6"
                    >
                        <v-card-title class="d-flex align-center">
                            <IconifyIcon
                                icon="mdi-account-star"
                                color="secondary"
                                class="mr-2"
                                aria-hidden="true"
                            ></IconifyIcon>
                            Famous Players
                        </v-card-title>
                        <v-card-text>
                            <v-chip-group
                                aria-label="Famous players who use this guitar"
                            >
                                <v-chip
                                    v-for="player in guitar.players"
                                    :key="player.id"
                                    :to="`/players/${player.id}`"
                                    color="primary"
                                    variant="outlined"
                                    class="ma-1"
                                >
                                    <v-avatar
                                        start
                                        size="24"
                                        aria-hidden="true"
                                    >
                                        <NuxtImg
                                            :src="
                                                player.image_url ||
                                                'https://via.placeholder.com/24'
                                            "
                                            :alt="player.name"
                                            width="24"
                                            height="24"
                                            loading="lazy"
                                            format="webp"
                                            quality="80"
                                        />
                                    </v-avatar>
                                    {{ player.name }}
                                </v-chip>
                            </v-chip-group>
                        </v-card-text>
                    </v-card>

                    <!-- Purchase Links -->
                    <PurchaseLinks
                        v-if="guitar.purchase_links"
                        :links="guitar.purchase_links"
                    />
                </v-col>
            </v-row>
        </template>
    </v-container>
</template>

<script setup lang="ts">
import type { Guitar } from "~/types";

const route = useRoute();
const { currentGuitar: guitar, loading, error, fetchGuitarById } = useGuitars();

const breadcrumbs = computed(() => [
    { title: "Home", to: "/", disabled: false },
    { title: "Guitars", to: "/guitars", disabled: false },
    { title: guitar.value?.model || "Loading...", disabled: true },
]);

const getTypeColor = (type: string) => {
    const colors: Record<string, string> = {
        electric: "red",
        acoustic: "green",
        bass: "blue",
    };
    return colors[type] || "grey";
};

const getTypeIcon = (type: string) => {
    const icons: Record<string, string> = {
        electric: "mdi-guitar-electric",
        acoustic: "mdi-guitar-acoustic",
        bass: "mdi-guitar-electric",
    };
    return icons[type] || "mdi-guitar";
};

const formatSpecKey = (key: string) => {
    return key.replace(/_/g, " ").replace(/\b\w/g, (l) => l.toUpperCase());
};

useHead({
    title: computed(() =>
        guitar.value
            ? `${guitar.value.brand?.name} ${guitar.value.model}`
            : "Guitar Details",
    ),
});

onMounted(async () => {
    const id = route.params.id as string;
    await fetchGuitarById(id);
});
</script>

<style scoped>
.guitar-detail-image {
    width: 100%;
    height: 500px;
    object-fit: cover;
    background-color: rgb(var(--v-theme-grey-lighten-3));
}

.guitar-detail-image-placeholder {
    width: 100%;
    height: 500px;
    padding: 2rem;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-weight: bold;
    font-size: 1.5rem;
    text-align: center;
}
</style>

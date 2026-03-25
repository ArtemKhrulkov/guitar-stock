<template>
    <v-card :to="`/guitars/${guitar.id}`" class="guitar-card h-100" hover>
        <NuxtImg
            v-if="guitar.image_url && !guitar.image_url.includes('placeholder')"
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
            <span>{{ guitar.brand?.name }} {{ guitar.model }}</span>
        </div>

        <!-- Type Badge -->
        <v-chip
            :color="getTypeColor(guitar.guitar_type)"
            size="small"
            class="type-badge"
        >
            {{ guitar.guitar_type }}
        </v-chip>

        <v-card-title class="pb-1">
            <span class="text-caption text-medium-emphasis">{{
                guitar.brand?.name
            }}</span>
            <div class="text-h6 font-weight-bold">{{ guitar.model }}</div>
        </v-card-title>

        <v-card-text>
            <!-- Price Range -->
            <div v-if="guitar.price_range" class="mb-3 flex items-center">
                <IconifyIcon
                    icon="mdi-currency-usd"
                    color="secondary"
                    class="mr-1"
                    aria-hidden="true"
                ></IconifyIcon>
                <span class="text-body-2 price-range">{{
                    guitar.price_range
                }}</span>
            </div>

            <!-- Specifications Preview -->
            <div
                v-if="guitar.specifications"
                class="text-caption text-medium-emphasis"
            >
                <div
                    v-if="guitar.specifications.body_wood"
                    class="flex items-center mb-2"
                >
                    <IconifyIcon
                        icon="mdi-cube-outline"
                        class="mr-1"
                        aria-hidden="true"
                    ></IconifyIcon>
                    {{ guitar.specifications.body_wood }}
                </div>
                <div
                    v-if="guitar.specifications.pickup_config"
                    class="flex items-center"
                >
                    <IconifyIcon
                        icon="mdi-tune"
                        class="mr-1"
                        aria-hidden="true"
                    ></IconifyIcon>
                    {{ guitar.specifications.pickup_config }}
                </div>
            </div>

            <!-- Famous Players -->
            <div
                v-if="guitar.players && guitar.players.length > 0"
                class="mt-2"
            >
                <PlayerBadge
                    v-for="player in guitar.players.slice(0, 2)"
                    :key="player.id"
                    :player="player"
                    class="mr-1 mb-1"
                />
            </div>
        </v-card-text>

        <v-card-actions>
            <v-btn color="primary" variant="text" block>
                View Details
                <IconifyIcon
                    icon="mdi-arrow-right"
                    class="ml-1"
                    aria-hidden="true"
                ></IconifyIcon>
            </v-btn>
        </v-card-actions>
    </v-card>
</template>

<script setup lang="ts">
import type { Guitar } from "~/types";

defineProps<{
    guitar: Guitar;
}>();

const getTypeColor = (type: string) => {
    const colors: Record<string, string> = {
        electric: "red",
        acoustic: "green",
        bass: "blue",
    };
    return colors[type] || "grey";
};
</script>

<style scoped>
.guitar-card {
    transition: all 0.3s ease;
}

.guitar-card:hover {
    transform: translateY(-4px);
}

.guitar-image {
    width: 100%;
    height: 200px;
    object-fit: cover;
    background-color: rgb(var(--v-theme-grey-lighten-3));
}

.guitar-image-placeholder {
    width: 100%;
    height: 200px;
    padding: 1rem;
    font-size: 0.9rem;
}

.type-badge {
    position: absolute;
    top: 8px;
    right: 8px;
}
</style>

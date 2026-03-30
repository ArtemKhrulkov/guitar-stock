<template>
    <Icon
        :icon="normalizedIcon"
        :width="sizeValue"
        :height="sizeValue"
        :color="colorValue"
        :class="className"
    />
</template>

<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { computed } from "vue";

const props = withDefaults(
    defineProps<{
        icon: string;
        size?: string | number;
        color?: string;
        class?: string;
    }>(),
    {
        size: "24",
        color: "currentColor",
    },
);

const sizeValue = computed(() => {
    const size = Number(props.size);
    return isNaN(size) ? props.size : size;
});

const colorValue = computed(() => props.color || "currentColor");

const className = computed(() => props.class || "");

const normalizedIcon = computed(() => {
    let icon = props.icon;
    if (!icon.includes(":")) {
        icon = `mdi:${icon.replace(/^mdi-/, "")}`;
    }
    return icon;
});
</script>

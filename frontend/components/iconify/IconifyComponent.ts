import { IconifyIcon } from "#components";
import { h } from "vue";
import type { IconProps } from "vuetify";

export const IconifyComponent = (props: IconProps) => {
  return h(IconifyIcon, {
    icon: props.icon || "mdi:help",
    width: (props as unknown as { icon?: string; size: string }).size || "24px",
  });
};

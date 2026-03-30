import { h } from "vue";
import { Icon } from "@iconify/vue";

export const IconifyComponent = (props: { icon?: string; size?: string }) => {
  let icon = props.icon || "mdi:help";
  if (!icon.includes(":")) {
    icon = `mdi:${icon.replace(/^mdi-/, "")}`;
  }
  
  return h(Icon, {
    icon,
    width: props.size || "24px",
    height: props.size || "24px",
  });
};

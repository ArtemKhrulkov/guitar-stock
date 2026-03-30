import { createVuetify } from "vuetify";
import * as components from "vuetify/components";
import * as directives from "vuetify/directives";
import { IconifyComponent } from "~/components/iconify/IconifyComponent";

export default defineNuxtPlugin((app) => {
  const vuetify = createVuetify({
    components,
    directives,
    icons: {
      defaultSet: "custom",
      sets: {
        custom: {
          component: IconifyComponent,
        },
      },
    },
    theme: {
      defaultTheme: "guitarDark",
      themes: {
        guitarDark: {
          dark: true,
          colors: {
            primary: "#9333EA",
            secondary: "#FFB300",
            accent: "#5D4037",
            error: "#FF5252",
            info: "#2196F3",
            success: "#4CAF50",
            warning: "#FFC107",
            background: "#121212",
            surface: "#1E1E1E",
          },
        },
        guitarLight: {
          dark: false,
          colors: {
            primary: "#9333EA",
            secondary: "#FFB300",
            accent: "#5D4037",
            error: "#FF5252",
            info: "#2196F3",
            success: "#4CAF50",
            warning: "#FFC107",
            background: "#FAFAFA",
            surface: "#FFFFFF",
          },
        },
      },
    },
    defaults: {
      VBtn: {
        variant: "flat",
        rounded: "lg",
      },
      VCard: {
        rounded: "lg",
        elevation: 2,
      },
      VTextField: {
        variant: "outlined",
        density: "comfortable",
      },
      VSelect: {
        variant: "outlined",
        density: "comfortable",
      },
    },
  });

  app.vueApp.use(vuetify);
});

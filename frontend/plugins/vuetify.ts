import { createVuetify } from 'vuetify';
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';
import { IconifyComponent } from '~/components/iconify/IconifyComponent';

export default defineNuxtPlugin((app) => {
  const vuetify = createVuetify({
    components,
    directives,
    icons: {
      defaultSet: 'custom',
      sets: {
        custom: {
          component: IconifyComponent,
        },
      },
    },
    theme: {
      defaultTheme: 'guitarDark',
      themes: {
        guitarDark: {
          dark: true,
          colors: {
            primary: '#9333EA',
            'primary-darken-1': '#7C3AED',
            secondary: '#FFB300',
            'secondary-darken-1': '#F59E0B',
            accent: '#5D4037',
            error: '#EF4444',
            info: '#3B82F6',
            success: '#22C55E',
            warning: '#F59E0B',
            background: '#0A0A0F',
            surface: '#1A1A24',
            'surface-bright': '#27273A',
            'surface-light': '#2D2D3F',
            'surface-variant': '#3F3F5C',
          },
        },
        guitarLight: {
          dark: false,
          colors: {
            primary: '#9333EA',
            'primary-darken-1': '#7C3AED',
            secondary: '#FFB300',
            'secondary-darken-1': '#F59E0B',
            accent: '#5D4037',
            error: '#EF4444',
            info: '#3B82F6',
            success: '#22C55E',
            warning: '#F59E0B',
            background: '#F8FAFC',
            surface: '#FFFFFF',
            'surface-bright': '#F1F5F9',
            'surface-light': '#E2E8F0',
            'surface-variant': '#CBD5E1',
          },
        },
      },
    },
    defaults: {
      VBtn: {
        variant: 'flat',
        rounded: 'lg',
      },
      VCard: {
        rounded: 'lg',
        elevation: 2,
      },
      VTextField: {
        variant: 'outlined',
        density: 'comfortable',
      },
      VSelect: {
        variant: 'outlined',
        density: 'comfortable',
      },
    },
  });

  app.vueApp.use(vuetify);
});

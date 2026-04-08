<template>
  <div class="auth-page">
    <v-container class="fill-height">
      <v-row justify="center" align="center">
        <v-col cols="12" sm="8" md="6" lg="4">
          <v-card class="pa-6" elevation="8">
            <div class="text-center mb-6">
              <IconifyIcon icon="mdi-guitar-acoustic" size="48" color="primary" />
              <h1 class="text-h5 mt-2">{{ isAdminMode ? 'Admin Login' : 'Login' }}</h1>
              <p class="text-body-2 text-medium-emphasis">
                {{ isAdminMode ? 'Enter admin credentials' : 'Sign in to your account' }}
              </p>
            </div>

            <v-form ref="formRef" @submit.prevent="handleLogin">
              <v-text-field
                v-model="form.email"
                label="Email"
                type="email"
                variant="outlined"
                prepend-inner-icon="mdi-email"
                :rules="[rules.required, rules.email]"
                :error-messages="errors.email"
                class="mb-4"
                :disabled="isAdminMode"
                :value="isAdminMode ? 'admin' : form.email"
                :hint="isAdminMode ? 'Default admin account' : undefined"
              />

              <v-text-field
                v-model="form.password"
                label="Password"
                :type="showPassword ? 'text' : 'password'"
                variant="outlined"
                prepend-inner-icon="mdi-lock"
                :append-inner-icon="showPassword ? 'mdi-eye-off' : 'mdi-eye'"
                :rules="[rules.required]"
                :error-messages="errors.password"
                class="mb-4"
                @click:append-inner="showPassword = !showPassword"
              />

              <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
                {{ error }}
              </v-alert>

              <v-btn
                type="submit"
                color="primary"
                block
                size="large"
                :loading="loading"
              >
                {{ isAdminMode ? 'Admin Login' : 'Login' }}
              </v-btn>
            </v-form>

            <div class="text-center mt-4">
              <NuxtLink v-if="!isAdminMode" to="/forgot-password" class="text-body-2">
                Forgot Password?
              </NuxtLink>
              <span v-if="!isAdminMode" class="mx-2">|</span>
              <NuxtLink v-if="!isAdminMode" to="/register" class="text-body-2">
                Register
              </NuxtLink>
            </div>

            <v-divider class="my-4" />

            <div class="text-center">
              <v-btn
                variant="text"
                size="small"
                @click="isAdminMode = !isAdminMode"
              >
                {{ isAdminMode ? 'Login as User' : 'Admin Login' }}
              </v-btn>
            </div>
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'default',
});

const { login } = useAuth();

const isAdminMode = ref(false);
const showPassword = ref(false);
const loading = ref(false);
const error = ref('');
const errors = ref<{ email?: string; password?: string }>({});

const form = ref({
  email: '',
  password: '',
});

const formRef = ref(null);

const rules = {
  required: (v: string) => !!v || 'Required',
  email: (v: string) => !isAdminMode.value ? (/.+@.+\..+/.test(v) || 'Invalid email') : true,
};

const handleLogin = async () => {
  errors.value = {};
  error.value = '';

  if (isAdminMode.value) {
    // Admin login - use hardcoded credentials from config
    form.value.email = 'admin';
  }

  if (!form.value.email && !isAdminMode.value) {
    errors.value.email = 'Email is required';
    return;
  }

  if (!form.value.password) {
    errors.value.password = 'Password is required';
    return;
  }

  loading.value = true;
  try {
    const user = await login(form.value.email, form.value.password);

    // Check if user is admin and redirect accordingly
    if (user.role === 'admin') {
      navigateTo('/admin');
    } else {
      navigateTo('/profile');
    }
  } catch (err: unknown) {
    const e = err as { data?: { error?: string } };
    error.value = e.data?.error || 'Login failed. Please check your credentials.';
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.auth-page {
  min-height: calc(100vh - 200px);
  display: flex;
  align-items: center;
}
</style>

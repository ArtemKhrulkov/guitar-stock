<template>
  <div class="auth-page">
    <v-container class="fill-height">
      <v-row justify="center" align="center">
        <v-col cols="12" sm="8" md="6" lg="4">
          <v-card class="pa-6" elevation="8">
            <div class="text-center mb-6">
              <IconifyIcon icon="mdi-guitar-acoustic" size="48" color="primary" />
              <h1 class="text-h5 mt-2">Create Account</h1>
              <p class="text-body-2 text-medium-emphasis">Join Guitar Stock to save your favorite guitars</p>
            </div>

            <v-form ref="formRef" @submit.prevent="handleRegister">
              <v-text-field
                v-model="form.email"
                label="Email"
                type="email"
                variant="outlined"
                prepend-inner-icon="mdi-email"
                :rules="[rules.required, rules.email]"
                :error-messages="emailError"
                class="mb-4"
              />

              <v-text-field
                v-model="form.name"
                label="Name (optional)"
                variant="outlined"
                prepend-inner-icon="mdi-account"
                class="mb-4"
              />

              <v-text-field
                v-model="form.password"
                label="Password"
                :type="showPassword ? 'text' : 'password'"
                variant="outlined"
                prepend-inner-icon="mdi-lock"
                :append-inner-icon="showPassword ? 'mdi-eye-off' : 'mdi-eye'"
                :rules="[rules.required, rules.minLength]"
                :error-messages="passwordError"
                class="mb-4"
                @click:append-inner="showPassword = !showPassword"
              />

              <v-text-field
                v-model="form.confirmPassword"
                label="Confirm Password"
                :type="showPassword ? 'text' : 'password'"
                variant="outlined"
                prepend-inner-icon="mdi-lock"
                :rules="[rules.required, rules.passwordMatch]"
                class="mb-4"
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
                Register
              </v-btn>
            </v-form>

            <div class="text-center mt-4">
              <span class="text-body-2 text-medium-emphasis">Already have an account?</span>
              <NuxtLink to="/login" class="ml-1">Login</NuxtLink>
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

const router = useRouter();
const { register } = useAuth();

const form = ref({
  email: '',
  name: '',
  password: '',
  confirmPassword: '',
});

const formRef = ref(null);

const loading = ref(false);
const error = ref('');
const emailError = ref('');
const passwordError = ref('');
const showPassword = ref(false);

const rules = {
  required: (v: string) => !!v || 'Required',
  email: (v: string) => /.+@.+\..+/.test(v) || 'Invalid email',
  minLength: (v: string) => v.length >= 8 || 'Min 8 characters',
  passwordMatch: (v: string) => v === form.value.password || 'Passwords must match',
};

const handleRegister = async () => {
  error.value = '';
  emailError.value = '';
  passwordError.value = '';

  if (form.value.password.length < 8) {
    passwordError.value = 'Password must be at least 8 characters';
    return;
  }

  if (form.value.password !== form.value.confirmPassword) {
    passwordError.value = 'Passwords do not match';
    return;
  }

  loading.value = true;
  try {
    await register(form.value.email, form.value.password, form.value.name || undefined);
    router.push({
      path: '/verify',
      query: { email: form.value.email, type: 'register' },
    });
  } catch (err: unknown) {
    const e = err as { data?: { error?: string } };
    error.value = e.data?.error || 'Registration failed. Please try again.';
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

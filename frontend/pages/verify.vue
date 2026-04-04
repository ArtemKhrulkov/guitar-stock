<template>
  <div class="auth-page">
    <v-container class="fill-height">
      <v-row justify="center" align="center">
        <v-col cols="12" sm="8" md="6" lg="4">
          <v-card class="pa-6" elevation="8">
            <div class="text-center mb-6">
              <IconifyIcon icon="mdi-email-check" size="48" color="primary" />
              <h1 class="text-h5 mt-2">Verify Email</h1>
              <p class="text-body-2 text-medium-emphasis">
                Enter the 6-digit code sent to<br>
                <strong>{{ email }}</strong>
              </p>
            </div>

            <div v-if="showResend" class="text-center mb-4">
              <p class="text-caption text-success">
                <IconifyIcon icon="mdi-check-circle" class="mr-1" />
                Code verified successfully!
              </p>
            </div>

            <v-form v-else @submit.prevent="handleVerify">
              <v-otp-input
                v-model="otp"
                length="6"
                type="number"
                variant="outlined"
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
                :disabled="otp.length !== 6"
              >
                Verify
              </v-btn>
            </v-form>

            <div class="text-center mt-4">
              <p v-if="countdown > 0" class="text-body-2 text-medium-emphasis">
                Resend code in {{ countdown }}s
              </p>
              <v-btn
                v-else
                variant="text"
                color="primary"
                @click="resendCode"
              >
                Resend Code
              </v-btn>
            </div>

            <div class="text-center mt-2">
              <NuxtLink to="/login" class="text-body-2">Back to Login</NuxtLink>
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

const route = useRoute();
const router = useRouter();
const { verifyEmail, requestPasswordReset } = useAuth();

const email = ref((route.query.email as string) || '');
const type = ref((route.query.type as string) || 'register');
const otp = ref('');
const loading = ref(false);
const error = ref('');
const countdown = ref(0);
const showResend = ref(false);

let countdownTimer: ReturnType<typeof setTimeout> | null = null;

const startCountdown = () => {
  countdown.value = 60;
  countdownTimer = setInterval(() => {
    countdown.value--;
    if (countdown.value <= 0 && countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = null;
    }
  }, 1000);
};

const handleVerify = async () => {
  if (otp.value.length !== 6) return;

  loading.value = true;
  error.value = '';

  try {
    await verifyEmail(email.value, otp.value);
    showResend.value = true;
    setTimeout(() => {
      router.push('/profile');
    }, 1500);
  } catch (err: unknown) {
    const e = err as { data?: { error?: string } };
    error.value = e.data?.error || 'Invalid or expired code';
    otp.value = '';
  } finally {
    loading.value = false;
  }
};

const resendCode = async () => {
  try {
    if (type.value === 'password_reset') {
      await requestPasswordReset(email.value);
    }
    startCountdown();
  } catch (err: unknown) {
    const e = err as { data?: { error?: string } };
    error.value = e.data?.error || 'Failed to resend code';
  }
};

onMounted(() => {
  if (!email.value) {
    router.push('/login');
  } else {
    startCountdown();
  }
});

onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer);
  }
});
</script>

<style scoped>
.auth-page {
  min-height: calc(100vh - 200px);
  display: flex;
  align-items: center;
}
</style>
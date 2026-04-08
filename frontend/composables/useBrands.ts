import type { Brand, Guitar } from '~/types';

export const useBrands = () => {
  const config = useRuntimeConfig();
  const apiUrl = config.public.apiUrl;

  const brands = ref<Brand[]>([]);
  const currentBrand = ref<Brand | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  const fetchBrands = async () => {
    loading.value = true;
    error.value = null;

    try {
      const response = await $fetch<{ brands: Brand[] }>(`${apiUrl}/brands`);
      brands.value = response.brands || [];
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch brands';
      console.error('Error fetching brands:', e);
    } finally {
      loading.value = false;
    }
  };

  const fetchBrandById = async (id: string) => {
    loading.value = true;
    error.value = null;

    try {
      const response = await $fetch<{ brand: Brand; guitars: Guitar[] }>(`${apiUrl}/brands/${id}`);
      currentBrand.value = response.brand;
      return response;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch brand';
      console.error('Error fetching brand:', e);
      return null;
    } finally {
      loading.value = false;
    }
  };

  return {
    brands,
    currentBrand,
    loading,
    error,
    fetchBrands,
    fetchBrandById,
  };
};

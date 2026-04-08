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
      const { data, error: fetchError } = await useAsyncData<{ brands: Brand[] }>(
        'brands-list',
        () => $fetch(`${apiUrl}/brands`),
        {
          default: () => ({ brands: [] }),
          lazy: true,
        },
      );

      if (fetchError.value) {
        error.value = fetchError.value.message || 'Failed to fetch brands';
      } else if (data.value) {
        brands.value = data.value.brands || [];
      }
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
      const { data, error: fetchError } = await useAsyncData<{ brand: Brand; guitars: Guitar[] }>(
        `brand-${id}`,
        () => $fetch(`${apiUrl}/brands/${id}`),
        {
          default: () => ({ brand: null as any, guitars: [] }),
          lazy: true,
        },
      );

      if (fetchError.value) {
        error.value = fetchError.value.message || 'Failed to fetch brand';
        return null;
      }

      if (data.value) {
        currentBrand.value = data.value.brand;
        return data.value;
      }

      return null;
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

import type { Guitar, GuitarFilters, PaginatedResponse, Player, PurchaseLink } from '~/types';

interface GuitarDetailResponse {
  guitar: Guitar;
  players: Player[];
  purchase_links: PurchaseLink[];
}

const buildQueryString = (filters: GuitarFilters = {}): string => {
  const queryParams = new URLSearchParams();

  if (filters.brands && filters.brands.length > 0) {
    filters.brands.forEach((brand) => queryParams.append('brand', brand));
  }
  if (filters.type) queryParams.set('type', filters.type);
  if (filters.search) queryParams.set('search', filters.search);
  if (filters.min_price) queryParams.set('min_price', filters.min_price.toString());
  if (filters.max_price) queryParams.set('max_price', filters.max_price.toString());
  if (filters.in_stock !== undefined) queryParams.set('in_stock', filters.in_stock.toString());
  if (filters.sort) queryParams.set('sort', filters.sort);
  if (filters.dir) queryParams.set('dir', filters.dir);
  if (filters.page) queryParams.set('page', filters.page.toString());
  if (filters.limit) queryParams.set('limit', filters.limit.toString());

  return queryParams.toString();
};

export const useGuitars = () => {
  const config = useRuntimeConfig();
  const apiUrl = config.public.apiUrl;

  const guitars = ref<Guitar[]>([]);
  const currentGuitar = ref<Guitar | null>(null);
  const total = ref(0);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const pending = ref(false);

  const currentFilters = ref<GuitarFilters>({});

  const setFilters = (filters: GuitarFilters) => {
    currentFilters.value = filters;
  };

  const fetchGuitars = async (filters: GuitarFilters = {}) => {
    loading.value = true;
    error.value = null;
    currentFilters.value = filters;

    try {
      const queryString = buildQueryString(filters);
      const response = await $fetch<PaginatedResponse<Guitar>>(
        `${apiUrl}/guitars?${queryString}`,
      );

      guitars.value = response.guitars || [];
      total.value = response.total;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch guitars';
      console.error('Error fetching guitars:', e);
    } finally {
      loading.value = false;
    }
  };

  const fetchGuitarById = async (id: string) => {
    loading.value = true;
    error.value = null;

    try {
      const response = await $fetch<GuitarDetailResponse>(`${apiUrl}/guitars/${id}`);
      currentGuitar.value = response.guitar;
      return response;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch guitar';
      console.error('Error fetching guitar:', e);
      return null;
    } finally {
      loading.value = false;
    }
  };

  return {
    guitars,
    currentGuitar,
    total,
    loading,
    error,
    pending,
    fetchGuitars,
    fetchGuitarById,
    setFilters,
  };
};

import type { Guitar, Player } from '~/types';

export interface SearchResults {
  guitars: Guitar[];
  players: Player[];
}

export const useSearch = () => {
  const config = useRuntimeConfig();
  const apiUrl = config.public.apiUrl;

  const results = ref<SearchResults>({ guitars: [], players: [] });
  const loading = ref(false);
  const error = ref<string | null>(null);
  const currentQuery = ref('');

  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  const search = async (query: string) => {
    currentQuery.value = query;

    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }

    if (!query.trim()) {
      results.value = { guitars: [], players: [] };
      return;
    }

    debounceTimer = setTimeout(async () => {
      loading.value = true;
      error.value = null;

      try {
        const { data, error: fetchError } = await useAsyncData<SearchResults>(
          `search-${query}`,
          () => $fetch(`${apiUrl}/search?q=${encodeURIComponent(query)}`),
          {
            default: () => ({ guitars: [], players: [] }),
            lazy: true,
          },
        );

        if (fetchError.value) {
          error.value = fetchError.value.message || 'Search failed';
        } else if (data.value && currentQuery.value === query) {
          results.value = data.value;
        }
      } catch (e) {
        error.value = e instanceof Error ? e.message : 'Search failed';
        console.error('Error searching:', e);
      } finally {
        loading.value = false;
      }
    }, 300);
  };

  const clearResults = () => {
    currentQuery.value = '';
    results.value = { guitars: [], players: [] };
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }
  };

  return {
    results,
    loading,
    error,
    search,
    clearResults,
  };
};

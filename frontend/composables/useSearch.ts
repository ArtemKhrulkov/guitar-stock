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
  const searchQuery = ref('');

  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  const search = async (query: string) => {
    searchQuery.value = query;

    if (!query.trim()) {
      results.value = { guitars: [], players: [] };
      return;
    }

    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }

    debounceTimer = setTimeout(async () => {
      loading.value = true;
      error.value = null;

      try {
        const response = await $fetch<SearchResults>(
          `${apiUrl}/search?q=${encodeURIComponent(query)}`,
        );
        results.value = response;
      } catch (e) {
        error.value = e instanceof Error ? e.message : 'Search failed';
        console.error('Error searching:', e);
      } finally {
        loading.value = false;
      }
    }, 300);
  };

  const clearResults = () => {
    searchQuery.value = '';
    results.value = { guitars: [], players: [] };
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }
  };

  return {
    results,
    loading,
    error,
    searchQuery,
    search,
    clearResults,
  };
};

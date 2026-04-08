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

  const search = async (query: string) => {
    searchQuery.value = query;

    if (!query.trim()) {
      results.value = { guitars: [], players: [] };
      return;
    }

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
  };

  const { data: searchData, pending: searchPending, refresh: refreshSearch } = useAsyncData(
    () => `search-${searchQuery.value}`,
    async () => {
      if (!searchQuery.value.trim()) {
        return { guitars: [], players: [] } as SearchResults;
      }
      const response = await $fetch<SearchResults>(
        `${apiUrl}/search?q=${encodeURIComponent(searchQuery.value)}`,
      );
      return response;
    },
    {
      default: () => ({ guitars: [], players: [] } as SearchResults),
      watch: [searchQuery],
    }
  );

  watch(searchData, (newData) => {
    if (newData) {
      results.value = newData;
    }
  });

  watch(searchPending, (isPending) => {
    loading.value = isPending;
  });

  const refresh = async () => {
    await refreshSearch();
  };

  const clearResults = () => {
    searchQuery.value = '';
    results.value = { guitars: [], players: [] };
  };

  return {
    results,
    loading,
    error,
    searchQuery,
    search,
    clearResults,
    refresh,
  };
};

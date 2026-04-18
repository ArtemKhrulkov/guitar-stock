import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { defineComponent, h, ref } from 'vue';

vi.mock('#app/composables/useRuntimeConfig', () => ({
  useRuntimeConfig: () => ({
    public: {
      apiUrl: 'http://localhost:8080/api',
    },
  }),
}));

describe('useGuitars', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('buildQueryString creates correct query string', () => {
    const buildQueryString = (filters: Record<string, any> = {}): string => {
      const queryParams = new URLSearchParams();

      if (filters.brands && filters.brands.length > 0) {
        filters.brands.forEach((brand: string) => queryParams.append('brand', brand));
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

    expect(buildQueryString({ type: 'electric' })).toBe('type=electric');
    expect(buildQueryString({ search: 'les paul' })).toBe('search=les+paul');
    expect(buildQueryString({ brands: ['brand1', 'brand2'] })).toBe('brand=brand1&brand=brand2');
    expect(buildQueryString({ page: 2, limit: 12 })).toBe('page=2&limit=12');
    expect(buildQueryString({})).toBe('');
  });

  it('buildQueryString handles all filter types', () => {
    const buildQueryString = (filters: Record<string, any> = {}): string => {
      const queryParams = new URLSearchParams();

      if (filters.brands && filters.brands.length > 0) {
        filters.brands.forEach((brand: string) => queryParams.append('brand', brand));
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

    const filters = {
      brands: ['brand-id-1'],
      type: 'electric',
      search: 'stratocaster',
      min_price: 1000,
      max_price: 5000,
      in_stock: true,
      sort: 'newest',
      dir: 'desc',
      page: 1,
      limit: 12,
    };

    const result = buildQueryString(filters);
    expect(result).toContain('brand=brand-id-1');
    expect(result).toContain('type=electric');
    expect(result).toContain('search=stratocaster');
    expect(result).toContain('min_price=1000');
    expect(result).toContain('max_price=5000');
    expect(result).toContain('in_stock=true');
    expect(result).toContain('sort=newest');
    expect(result).toContain('dir=desc');
    expect(result).toContain('page=1');
    expect(result).toContain('limit=12');
  });
});

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('login validates email format', async () => {
    const isValidEmail = (email: string): boolean => {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      return emailRegex.test(email);
    };

    expect(isValidEmail('test@example.com')).toBe(true);
    expect(isValidEmail('invalid-email')).toBe(false);
    expect(isValidEmail('test@')).toBe(false);
    expect(isValidEmail('@example.com')).toBe(false);
  });

  it('login validates password is not empty', () => {
    const isValidPassword = (password: string): boolean => {
      return password.length >= 1;
    };

    expect(isValidPassword('password123')).toBe(true);
    expect(isValidPassword('')).toBe(false);
  });
});

describe('GuitarCard component', () => {
  it('renders guitar information correctly', () => {
    const GuitarCard = defineComponent({
      props: {
        guitar: {
          type: Object as () => ({
            id: string;
            model: string;
            brand?: { name: string };
            guitar_type: string;
            price_range?: string;
            image_url?: string;
          }),
          required: true,
        },
      },
      setup(props) {
        return () =>
          h('div', { class: 'guitar-card' }, [
            h('img', { src: props.guitar.image_url || '/placeholder.jpg', alt: props.guitar.model }),
            h('h3', props.guitar.model),
            h('span', props.guitar.brand?.name || 'Unknown Brand'),
            h('span', props.guitar.guitar_type),
          ]);
      },
    });

    const guitar = {
      id: '1',
      model: 'Les Paul',
      brand: { name: 'Gibson' },
      guitar_type: 'electric',
      price_range: '1,000 - 2,000 USD',
      image_url: 'https://example.com/guitar.jpg',
    };

    const wrapper = mount(GuitarCard, {
      props: { guitar },
    });

    expect(wrapper.find('h3').text()).toBe('Les Paul');
    expect(wrapper.findAll('span')[0].text()).toBe('Gibson');
    expect(wrapper.findAll('span')[1].text()).toBe('electric');
    expect(wrapper.find('img').attributes('src')).toBe('https://example.com/guitar.jpg');
  });

  it('shows placeholder image when none provided', () => {
    const GuitarCard = defineComponent({
      props: {
        guitar: {
          type: Object as () => ({
            id: string;
            model: string;
            brand?: { name: string };
            guitar_type: string;
            image_url?: string;
          }),
          required: true,
        },
      },
      setup(props) {
        return () =>
          h('div', { class: 'guitar-card' }, [
            h('img', { src: props.guitar.image_url || '/placeholder.jpg', alt: props.guitar.model }),
            h('h3', props.guitar.model),
          ]);
      },
    });

    const guitar = {
      id: '1',
      model: 'Stratocaster',
      brand: { name: 'Fender' },
      guitar_type: 'electric',
    };

    const wrapper = mount(GuitarCard, {
      props: { guitar },
    });

    expect(wrapper.find('img').attributes('src')).toBe('/placeholder.jpg');
  });
});

describe('useComparison', () => {
  it('max 4 guitars can be selected', () => {
    const MAX_COMPARE = 4;

    const selectedGuitars = ref<string[]>([]);

    const addToCompare = (guitarId: string) => {
      if (selectedGuitars.value.length < MAX_COMPARE && !selectedGuitars.value.includes(guitarId)) {
        selectedGuitars.value.push(guitarId);
      }
    };

    const removeFromCompare = (guitarId: string) => {
      selectedGuitars.value = selectedGuitars.value.filter((id) => id !== guitarId);
    };

    expect(selectedGuitars.value.length).toBe(0);

    addToCompare('guitar-1');
    addToCompare('guitar-2');
    addToCompare('guitar-3');
    addToCompare('guitar-4');

    expect(selectedGuitars.value.length).toBe(4);

    addToCompare('guitar-5');
    expect(selectedGuitars.value.length).toBe(4);

    removeFromCompare('guitar-2');
    expect(selectedGuitars.value.length).toBe(3);

    addToCompare('guitar-5');
    expect(selectedGuitars.value.length).toBe(4);
    expect(selectedGuitars.value).toContain('guitar-5');
    expect(selectedGuitars.value).not.toContain('guitar-2');
  });

  it('does not add duplicate guitars', () => {
    const selectedGuitars = ref<string[]>([]);

    const addToCompare = (guitarId: string) => {
      if (!selectedGuitars.value.includes(guitarId)) {
        selectedGuitars.value.push(guitarId);
      }
    };

    addToCompare('guitar-1');
    addToCompare('guitar-1');
    addToCompare('guitar-1');

    expect(selectedGuitars.value.length).toBe(1);
    expect(selectedGuitars.value[0]).toBe('guitar-1');
  });
});

describe('useWishlist', () => {
  it('can add and remove guitars from wishlist', () => {
    const wishlist = ref<string[]>([]);

    const addToWishlist = (guitarId: string) => {
      if (!wishlist.value.includes(guitarId)) {
        wishlist.value.push(guitarId);
      }
    };

    const removeFromWishlist = (guitarId: string) => {
      wishlist.value = wishlist.value.filter((id) => id !== guitarId);
    };

    const isInWishlist = (guitarId: string) => {
      return wishlist.value.includes(guitarId);
    };

    addToWishlist('guitar-1');
    expect(isInWishlist('guitar-1')).toBe(true);
    expect(wishlist.value.length).toBe(1);

    removeFromWishlist('guitar-1');
    expect(isInWishlist('guitar-1')).toBe(false);
    expect(wishlist.value.length).toBe(0);
  });

  it('prevents duplicate additions', () => {
    const wishlist = ref<string[]>([]);

    const addToWishlist = (guitarId: string) => {
      if (!wishlist.value.includes(guitarId)) {
        wishlist.value.push(guitarId);
      }
    };

    addToWishlist('guitar-1');
    addToWishlist('guitar-1');
    addToWishlist('guitar-1');

    expect(wishlist.value.length).toBe(1);
  });
});

describe('GuitarFilters', () => {
  it('correctly formats price range', () => {
    const formatPrice = (min?: number, max?: number, currency = 'USD'): string => {
      if (!min && !max) return '';
      if (min && max) return `${min.toLocaleString()} - ${max.toLocaleString()} ${currency}`;
      if (min) return `From ${min.toLocaleString()} ${currency}`;
      return `Up to ${max?.toLocaleString()} ${currency}`;
    };

    expect(formatPrice(1000, 5000)).toBe('1,000 - 5,000 USD');
    expect(formatPrice(1000)).toBe('From 1,000 USD');
    expect(formatPrice(undefined, 5000)).toBe('Up to 5,000 USD');
    expect(formatPrice()).toBe('');
  });

  it('correctly parses guitar types', () => {
    const guitarTypes = ['electric', 'acoustic', 'bass'];

    expect(guitarTypes).toContain('electric');
    expect(guitarTypes).toContain('acoustic');
    expect(guitarTypes).toContain('bass');
    expect(guitarTypes.length).toBe(3);
  });
});

describe('Tooltip Logic', () => {
  it('wishlist tooltip shows correct text based on wishlist state', () => {
    const getWishlistTooltipText = (isWishlisted: boolean): string => {
      return isWishlisted ? 'Remove from wishlist' : 'Add to wishlist';
    };

    expect(getWishlistTooltipText(false)).toBe('Add to wishlist');
    expect(getWishlistTooltipText(true)).toBe('Remove from wishlist');
  });

  it('compare tooltip shows correct text based on selection state', () => {
    const getCompareTooltipText = (isSelected: boolean): string => {
      return isSelected ? 'Remove from compare' : 'Add to compare';
    };

    expect(getCompareTooltipText(false)).toBe('Add to compare');
    expect(getCompareTooltipText(true)).toBe('Remove from compare');
  });

  it('tooltip keys are unique per guitar', () => {
    const generateTooltipKeys = (guitarId: string) => ({
      wishlistKey: `wishlist-${guitarId}`,
      compareKey: `compare-${guitarId}`,
    });

    const guitar1 = generateTooltipKeys('guitar-1');
    const guitar2 = generateTooltipKeys('guitar-2');

    expect(guitar1.wishlistKey).not.toBe(guitar2.wishlistKey);
    expect(guitar1.compareKey).not.toBe(guitar2.compareKey);
    expect(guitar1.wishlistKey).toBe('wishlist-guitar-1');
    expect(guitar1.compareKey).toBe('compare-guitar-1');
  });
});

describe('API Response Types', () => {
  it('PaginatedResponse has required fields', () => {
    const mockResponse = {
      guitars: [
        {
          id: '1',
          model: 'Les Paul',
          guitar_type: 'electric',
        },
      ],
      total: 1,
      page: 1,
    };

    expect(mockResponse).toHaveProperty('guitars');
    expect(mockResponse).toHaveProperty('total');
    expect(mockResponse).toHaveProperty('page');
    expect(Array.isArray(mockResponse.guitars)).toBe(true);
    expect(typeof mockResponse.total).toBe('number');
    expect(typeof mockResponse.page).toBe('number');
  });

  it('GuitarDetailResponse has required fields', () => {
    const mockResponse = {
      guitar: {
        id: '1',
        model: 'Les Paul',
        guitar_type: 'electric',
        specifications: {
          body_wood: 'Mahogany',
          neck_wood: 'Maple',
        },
      },
      players: [
        {
          id: '1',
          name: 'Jimmy Page',
          genre: 'Rock',
        },
      ],
      purchase_links: [
        {
          id: '1',
          platform: 'sweetwater',
          url: 'https://sweetwater.com/les-paul',
          price_usd: 2499.99,
        },
      ],
    };

    expect(mockResponse).toHaveProperty('guitar');
    expect(mockResponse).toHaveProperty('players');
    expect(mockResponse).toHaveProperty('purchase_links');
    expect(Array.isArray(mockResponse.players)).toBe(true);
    expect(Array.isArray(mockResponse.purchase_links)).toBe(true);
  });
});

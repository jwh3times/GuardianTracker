import {
  formatDate,
  formatDateTime,
  getRarityColor,
  getDifficultyColor,
  getPriorityColor,
  sortItemsByDifficulty,
  sortItemsByRarity,
  filterItemsByDifficulty,
  filterItemsByRarity,
  calculateCollectionProgress,
  getItemTypeIcon,
  formatNumber,
  validateImageUrl,
} from '../lib/utils';
import { DestinyItem } from '../types';

// Mock item factory
const createMockItem = (overrides: Partial<DestinyItem> = {}): DestinyItem => ({
  itemHash: '12345',
  name: 'Test Item',
  description: 'A test item',
  icon: '/icons/test.png',
  rarity: 'Legendary',
  tierType: 6,
  itemType: 'Auto Rifle',
  difficulty: 'Moderate',
  isExotic: false,
  sources: ['Source 1'],
  ...overrides,
});

describe('Utility Functions', () => {
  describe('formatDate', () => {
    it('should format date correctly', () => {
      const result = formatDate('2024-01-15T10:30:00Z');
      expect(result).toMatch(/Jan 15, 2024/);
    });
  });

  describe('formatDateTime', () => {
    it('should format datetime correctly', () => {
      const result = formatDateTime('2024-01-15T10:30:00Z');
      expect(result).toContain('Jan');
      expect(result).toContain('15');
      expect(result).toContain('2024');
    });
  });

  describe('getRarityColor', () => {
    it('should return correct color for exotic', () => {
      expect(getRarityColor('exotic')).toBe('text-destiny-exotic');
      expect(getRarityColor('Exotic')).toBe('text-destiny-exotic');
    });

    it('should return correct color for legendary', () => {
      expect(getRarityColor('legendary')).toBe('text-destiny-legendary');
    });

    it('should return correct color for rare', () => {
      expect(getRarityColor('rare')).toBe('text-destiny-rare');
    });

    it('should return default for unknown rarity', () => {
      expect(getRarityColor('unknown')).toBe('text-foreground');
    });
  });

  describe('getDifficultyColor', () => {
    it('should return green for easy', () => {
      expect(getDifficultyColor('easy')).toBe('text-green-400');
      expect(getDifficultyColor('Easy')).toBe('text-green-400');
    });

    it('should return yellow for moderate', () => {
      expect(getDifficultyColor('moderate')).toBe('text-yellow-400');
    });

    it('should return red for challenging', () => {
      expect(getDifficultyColor('challenging')).toBe('text-red-400');
    });

    it('should return default for unknown difficulty', () => {
      expect(getDifficultyColor('unknown')).toBe('text-foreground');
    });
  });

  describe('getPriorityColor', () => {
    it('should return correct colors for priorities', () => {
      expect(getPriorityColor('URGENT')).toContain('red');
      expect(getPriorityColor('HIGH')).toContain('orange');
      expect(getPriorityColor('MEDIUM')).toContain('yellow');
      expect(getPriorityColor('LOW')).toContain('green');
    });
  });

  describe('sortItemsByDifficulty', () => {
    it('should sort items by difficulty (Easy first)', () => {
      const items = [
        createMockItem({ name: 'Challenging', difficulty: 'Challenging' }),
        createMockItem({ name: 'Easy', difficulty: 'Easy' }),
        createMockItem({ name: 'Moderate', difficulty: 'Moderate' }),
      ];

      const sorted = sortItemsByDifficulty(items);

      expect(sorted[0].difficulty).toBe('Easy');
      expect(sorted[1].difficulty).toBe('Moderate');
      expect(sorted[2].difficulty).toBe('Challenging');
    });

    it('should not mutate original array', () => {
      const items = [
        createMockItem({ difficulty: 'Challenging' }),
        createMockItem({ difficulty: 'Easy' }),
      ];
      const original = [...items];

      sortItemsByDifficulty(items);

      expect(items).toEqual(original);
    });
  });

  describe('sortItemsByRarity', () => {
    it('should sort items by rarity (Exotic first)', () => {
      const items = [
        createMockItem({ name: 'Common', rarity: 'Common' }),
        createMockItem({ name: 'Exotic', rarity: 'Exotic' }),
        createMockItem({ name: 'Legendary', rarity: 'Legendary' }),
      ];

      const sorted = sortItemsByRarity(items);

      expect(sorted[0].rarity).toBe('Exotic');
      expect(sorted[1].rarity).toBe('Legendary');
      expect(sorted[2].rarity).toBe('Common');
    });
  });

  describe('filterItemsByDifficulty', () => {
    it('should filter items by specified difficulties', () => {
      const items = [
        createMockItem({ difficulty: 'Easy' }),
        createMockItem({ difficulty: 'Moderate' }),
        createMockItem({ difficulty: 'Challenging' }),
      ];

      const filtered = filterItemsByDifficulty(items, ['Easy', 'Moderate']);

      expect(filtered).toHaveLength(2);
      expect(filtered.every(item => ['Easy', 'Moderate'].includes(item.difficulty))).toBe(true);
    });

    it('should return all items when filter is empty', () => {
      const items = [
        createMockItem({ difficulty: 'Easy' }),
        createMockItem({ difficulty: 'Challenging' }),
      ];

      const filtered = filterItemsByDifficulty(items, []);

      expect(filtered).toHaveLength(2);
    });
  });

  describe('filterItemsByRarity', () => {
    it('should filter items by specified rarities', () => {
      const items = [
        createMockItem({ rarity: 'Common' }),
        createMockItem({ rarity: 'Exotic' }),
        createMockItem({ rarity: 'Legendary' }),
      ];

      const filtered = filterItemsByRarity(items, ['Exotic']);

      expect(filtered).toHaveLength(1);
      expect(filtered[0].rarity).toBe('Exotic');
    });
  });

  describe('calculateCollectionProgress', () => {
    it('should calculate percentage correctly', () => {
      expect(calculateCollectionProgress(100, 50)).toBe(50);
      expect(calculateCollectionProgress(200, 150)).toBe(75);
      expect(calculateCollectionProgress(10, 10)).toBe(100);
    });

    it('should return 0 when total is 0', () => {
      expect(calculateCollectionProgress(0, 0)).toBe(0);
    });

    it('should round to nearest integer', () => {
      expect(calculateCollectionProgress(3, 1)).toBe(33);
    });
  });

  describe('getItemTypeIcon', () => {
    it('should return correct icon for known item types', () => {
      expect(getItemTypeIcon('Sword')).toBe('⚔️');
      expect(getItemTypeIcon('Rocket Launcher')).toBe('🚀');
    });

    it('should return default icon for unknown types', () => {
      expect(getItemTypeIcon('Unknown Type')).toBe('📦');
    });
  });

  describe('formatNumber', () => {
    it('should format millions correctly', () => {
      expect(formatNumber(1500000)).toBe('1.5M');
      expect(formatNumber(1000000)).toBe('1.0M');
    });

    it('should format thousands correctly', () => {
      expect(formatNumber(1500)).toBe('1.5K');
      expect(formatNumber(1000)).toBe('1.0K');
    });

    it('should return number as string for small values', () => {
      expect(formatNumber(500)).toBe('500');
      expect(formatNumber(0)).toBe('0');
    });
  });

  describe('validateImageUrl', () => {
    it('should return placeholder for empty url', () => {
      expect(validateImageUrl('')).toBe('/placeholder-item.png');
    });

    it('should return url as-is if it starts with http', () => {
      const url = 'https://example.com/image.png';
      expect(validateImageUrl(url)).toBe(url);
    });

    it('should prepend bungie.net for relative urls', () => {
      const url = '/icons/item.png';
      expect(validateImageUrl(url)).toBe('https://www.bungie.net/icons/item.png');
    });
  });
});

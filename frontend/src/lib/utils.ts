import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"
import { DestinyItem, WishListPriority } from '../types';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function formatDateTime(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function getTimeUntilReset(): string {
  const now = new Date();
  const nextTuesday = new Date();
  
  // Find next Tuesday at 10 AM PST (Destiny 2 weekly reset)
  nextTuesday.setDate(now.getDate() + (2 - now.getDay() + 7) % 7);
  nextTuesday.setHours(17, 0, 0, 0); // 10 AM PST = 17:00 UTC
  
  if (nextTuesday <= now) {
    nextTuesday.setDate(nextTuesday.getDate() + 7);
  }
  
  const timeUntil = nextTuesday.getTime() - now.getTime();
  const days = Math.floor(timeUntil / (1000 * 60 * 60 * 24));
  const hours = Math.floor((timeUntil % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const minutes = Math.floor((timeUntil % (1000 * 60 * 60)) / (1000 * 60));
  
  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`;
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else {
    return `${minutes}m`;
  }
}

export function getRarityColor(rarity: string): string {
  switch (rarity.toLowerCase()) {
    case 'exotic':
      return 'text-destiny-exotic';
    case 'legendary':
      return 'text-destiny-legendary';
    case 'rare':
      return 'text-destiny-rare';
    case 'uncommon':
      return 'text-destiny-uncommon';
    case 'common':
      return 'text-destiny-common';
    default:
      return 'text-foreground';
  }
}

export function getDifficultyColor(difficulty: string): string {
  switch (difficulty.toLowerCase()) {
    case 'easy':
      return 'text-green-400';
    case 'moderate':
      return 'text-yellow-400';
    case 'challenging':
      return 'text-red-400';
    default:
      return 'text-foreground';
  }
}

export function getPriorityColor(priority: WishListPriority): string {
  switch (priority) {
    case 'URGENT':
      return 'text-red-400 border-red-400';
    case 'HIGH':
      return 'text-orange-400 border-orange-400';
    case 'MEDIUM':
      return 'text-yellow-400 border-yellow-400';
    case 'LOW':
      return 'text-green-400 border-green-400';
    default:
      return 'text-foreground border-border';
  }
}

export function sortItemsByDifficulty(items: DestinyItem[]): DestinyItem[] {
  const difficultyOrder = { 'Easy': 1, 'Moderate': 2, 'Challenging': 3 };
  return [...items].sort((a, b) => {
    const orderA = difficultyOrder[a.difficulty] || 4;
    const orderB = difficultyOrder[b.difficulty] || 4;
    return orderA - orderB;
  });
}

export function sortItemsByRarity(items: DestinyItem[]): DestinyItem[] {
  const rarityOrder = { 'Common': 1, 'Uncommon': 2, 'Rare': 3, 'Legendary': 4, 'Exotic': 5 };
  return [...items].sort((a, b) => {
    const orderA = rarityOrder[a.rarity] || 6;
    const orderB = rarityOrder[b.rarity] || 6;
    return orderB - orderA; // Reverse order (Exotic first)
  });
}

export function filterItemsByDifficulty(items: DestinyItem[], difficulties: string[]): DestinyItem[] {
  if (difficulties.length === 0) return items;
  return items.filter(item => difficulties.includes(item.difficulty));
}

export function filterItemsByRarity(items: DestinyItem[], rarities: string[]): DestinyItem[] {
  if (rarities.length === 0) return items;
  return items.filter(item => rarities.includes(item.rarity));
}

export function calculateCollectionProgress(total: number, collected: number): number {
  if (total === 0) return 0;
  return Math.round((collected / total) * 100);
}

export function getItemTypeIcon(itemType: string): string {
  // Map item types to icons - using placeholder for now
  const iconMap: Record<string, string> = {
    'Auto Rifle': '🔫',
    'Hand Cannon': '🔫',
    'Scout Rifle': '🔫',
    'Pulse Rifle': '🔫',
    'Submachine Gun': '🔫',
    'Shotgun': '💥',
    'Sniper Rifle': '🎯',
    'Fusion Rifle': '⚡',
    'Rocket Launcher': '🚀',
    'Grenade Launcher': '💣',
    'Sword': '⚔️',
    'Helmet': '⛑️',
    'Gauntlets': '🧤',
    'Chest Armor': '🛡️',
    'Leg Armor': '👢',
    'Class Armor': '🎯',
  };
  
  return iconMap[itemType] || '📦';
}

export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout;
  return (...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}

export function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M';
  } else if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K';
  }
  return num.toString();
}

export function validateImageUrl(url: string): string {
  if (!url) return '/placeholder-item.png';
  if (url.startsWith('http')) return url;
  return `https://www.bungie.net${url}`;
}

import React, { useState } from 'react';
import { useQuery } from '@apollo/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/Card';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import { Button } from '../components/ui/Button';
import { GET_CURRENT_USER, GET_USER_COLLECTIONS } from '../graphql/queries';
import { DestinyItem } from '../types';
import { getRarityColor, getDifficultyColor, validateImageUrl, sortItemsByDifficulty } from '../lib/utils';

export function Collections() {
  const [activeTab, setActiveTab] = useState<'weapons' | 'armor' | 'exotics'>('weapons');
  const [difficultyFilter, setDifficultyFilter] = useState<string[]>([]);

  const { data: userData } = useQuery(GET_CURRENT_USER);
  const { data: collectionsData, loading } = useQuery(GET_USER_COLLECTIONS, {
    variables: {
      membershipType: userData?.currentUser?.membershipType,
      membershipId: userData?.currentUser?.membershipId,
    },
    skip: !userData?.currentUser,
  });

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  const collections = collectionsData?.userCollections;
  const activeCollection = collections?.[activeTab];
  
  const filteredItems = difficultyFilter.length > 0
    ? activeCollection?.missing?.filter((item: DestinyItem) => 
        difficultyFilter.includes(item.difficulty)
      ) || []
    : activeCollection?.missing || [];

  const sortedItems = sortItemsByDifficulty(filteredItems);

  const handleDifficultyFilter = (difficulty: string) => {
    setDifficultyFilter(prev => 
      prev.includes(difficulty)
        ? prev.filter(d => d !== difficulty)
        : [...prev, difficulty]
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Collections</h1>
        <p className="text-muted-foreground">
          Track your missing items and plan your acquisitions
        </p>
      </div>

      {/* Collection Tabs */}
      <div className="flex space-x-4 border-b">
        {[
          { key: 'weapons', label: 'Weapons', icon: '⚔️' },
          { key: 'armor', label: 'Armor', icon: '🛡️' },
          { key: 'exotics', label: 'Exotics', icon: '✨' },
        ].map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key as any)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab.key
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <span className="mr-2">{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle>Filters</CardTitle>
          <CardDescription>Filter items by acquisition difficulty</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {['Easy', 'Moderate', 'Challenging'].map((difficulty) => (
              <Button
                key={difficulty}
                variant={difficultyFilter.includes(difficulty) ? 'default' : 'outline'}
                size="sm"
                onClick={() => handleDifficultyFilter(difficulty)}
                className={getDifficultyColor(difficulty)}
              >
                {difficulty}
              </Button>
            ))}
            {difficultyFilter.length > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setDifficultyFilter([])}
              >
                Clear Filters
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Collection Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">
              <div className="text-2xl font-bold">{activeCollection?.total || 0}</div>
              <p className="text-sm text-muted-foreground">Total Items</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-green-400">{activeCollection?.collected || 0}</div>
              <p className="text-sm text-muted-foreground">Collected</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-red-400">{activeCollection?.missing?.length || 0}</div>
              <p className="text-sm text-muted-foreground">Missing</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Items Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {sortedItems.map((item: DestinyItem) => (
          <Card key={item.itemHash} className={`destiny-card hover:shadow-lg transition-shadow ${
            item.isExotic ? 'destiny-card-exotic' : 
            item.rarity === 'Legendary' ? 'destiny-card-legendary' : 
            'destiny-card-rare'
          }`}>
            <CardContent className="p-4">
              <div className="space-y-3">
                <div className="flex items-start space-x-3">
                  <img
                    src={validateImageUrl(item.icon)}
                    alt={item.name}
                    className="w-12 h-12 rounded-lg"
                  />
                  <div className="flex-1 min-w-0">
                    <h3 className={`font-semibold text-sm ${getRarityColor(item.rarity)}`}>
                      {item.name}
                    </h3>
                    <p className="text-xs text-muted-foreground">{item.itemType}</p>
                  </div>
                </div>
                
                <p className="text-xs text-muted-foreground line-clamp-2">
                  {item.description}
                </p>
                
                <div className="flex items-center justify-between text-xs">
                  <span className={getDifficultyColor(item.difficulty)}>
                    {item.difficulty}
                  </span>
                  <span className={getRarityColor(item.rarity)}>
                    {item.rarity}
                  </span>
                </div>
                
                {item.sources && item.sources.length > 0 && (
                  <div className="text-xs text-muted-foreground">
                    <p className="font-medium">Sources:</p>
                    <ul className="list-disc list-inside space-y-1">
                      {item.sources.slice(0, 2).map((source, index) => (
                        <li key={index}>{source}</li>
                      ))}
                      {item.sources.length > 2 && (
                        <li>+{item.sources.length - 2} more</li>
                      )}
                    </ul>
                  </div>
                )}
                
                <Button 
                  size="sm" 
                  className="w-full"
                  variant="outline"
                >
                  Add to Wish List
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {sortedItems.length === 0 && (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="text-6xl mb-4">🎉</div>
            <h3 className="text-lg font-semibold mb-2">All Caught Up!</h3>
            <p className="text-muted-foreground">
              {difficultyFilter.length > 0
                ? 'No items match the current filters.'
                : 'You have collected all items in this category.'}
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

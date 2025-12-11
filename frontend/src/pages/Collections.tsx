import React, { useState } from 'react';
import { useQuery, useMutation } from '@apollo/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/Card';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import { Button } from '../components/ui/Button';
import { GET_CURRENT_USER, GET_USER_COLLECTIONS, GET_WISH_LIST } from '../graphql/queries';
import { ADD_TO_WISH_LIST } from '../graphql/mutations';
import { DestinyItem } from '../types';
import { getRarityColor, getDifficultyColor, validateImageUrl, sortItemsByDifficulty } from '../lib/utils';
import { useToast } from '../components/ui/Toast';

type CollectionTab = 'weapons' | 'armor' | 'exotics';

// Debug banner component for testing
function DataSourceBanner({
  collections,
  loading,
  error,
  onRefresh
}: {
  collections: any;
  loading: boolean;
  error: any;
  onRefresh: () => void;
}) {
  // Determine data source based on data characteristics
  const isRealData = collections && (
    collections.weapons?.total > 0 ||
    collections.armor?.total > 0 ||
    collections.exotics?.total > 0
  );

  const hasItems = collections && (
    (collections.weapons?.missing?.length > 0) ||
    (collections.armor?.missing?.length > 0) ||
    (collections.exotics?.missing?.length > 0)
  );

  // Check if items have real Bungie data (icons start with /common/)
  const hasRealIcons = collections?.weapons?.missing?.[0]?.icon?.startsWith('/common/') ||
    collections?.armor?.missing?.[0]?.icon?.startsWith('/common/') ||
    collections?.exotics?.missing?.[0]?.icon?.startsWith('/common/');

  if (loading) return null;

  return (
    <div className={`rounded-lg p-3 mb-4 flex items-center justify-between ${
      error
        ? 'bg-red-500/20 border border-red-500/50'
        : isRealData && hasRealIcons
          ? 'bg-green-500/20 border border-green-500/50'
          : 'bg-yellow-500/20 border border-yellow-500/50'
    }`}>
      <div className="flex items-center gap-2">
        <span className="text-lg">
          {error ? '❌' : isRealData && hasRealIcons ? '✅' : '⚠️'}
        </span>
        <div>
          <p className="font-medium text-sm">
            {error
              ? 'Connection Error'
              : isRealData && hasRealIcons
                ? 'Live Bungie Data'
                : 'Mock/Test Data'}
          </p>
          <p className="text-xs text-muted-foreground">
            {error
              ? `Error: ${error.message}`
              : isRealData && hasRealIcons
                ? `Showing ${collections.weapons?.total + collections.armor?.total + collections.exotics?.total} real collectibles`
                : hasItems
                  ? 'Data loaded but may be mock data'
                  : 'No collection data available'}
          </p>
        </div>
      </div>
      <Button
        variant="outline"
        size="sm"
        onClick={onRefresh}
        disabled={loading}
      >
        {loading ? <LoadingSpinner size="sm" /> : 'Refresh'}
      </Button>
    </div>
  );
}

export function Collections() {
  const [activeTab, setActiveTab] = useState<CollectionTab>('weapons');
  const [difficultyFilter, setDifficultyFilter] = useState<string[]>([]);
  const [addingItems, setAddingItems] = useState<Set<string>>(new Set());
  const { showToast } = useToast();

  const { data: userData } = useQuery(GET_CURRENT_USER);
  const { data: collectionsData, loading, error, refetch } = useQuery(GET_USER_COLLECTIONS, {
    variables: {
      membershipType: userData?.currentUser?.membershipType,
      membershipId: userData?.currentUser?.membershipId,
    },
    skip: !userData?.currentUser,
    fetchPolicy: 'cache-and-network', // Always check server for fresh data
  });

  const handleRefresh = () => {
    refetch();
  };

  const [addToWishList] = useMutation(ADD_TO_WISH_LIST, {
    refetchQueries: [{ query: GET_WISH_LIST }],
    onCompleted: () => {
      showToast('Item added to wish list!', 'success');
    },
    onError: (err) => {
      showToast(`Failed to add item: ${err.message}`, 'error');
    },
  });

  const handleAddToWishList = async (item: DestinyItem) => {
    if (addingItems.has(item.itemHash)) return;

    setAddingItems(prev => new Set(prev).add(item.itemHash));

    try {
      await addToWishList({
        variables: {
          itemHash: item.itemHash,
          priority: 'MEDIUM',
        },
      });
    } finally {
      setAddingItems(prev => {
        const next = new Set(prev);
        next.delete(item.itemHash);
        return next;
      });
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (error) {
    // Parse error for better display
    const errorCode = error.graphQLErrors?.[0]?.extensions?.code || 'UNKNOWN';
    const isAuthError = error.message.includes('token') || error.message.includes('auth') || error.message.includes('401');
    const isBungieError = error.message.includes('Bungie') || errorCode === 'BUNGIE_ERROR';

    return (
      <div className="space-y-6">
        <DataSourceBanner
          collections={null}
          loading={false}
          error={error}
          onRefresh={handleRefresh}
        />
        <Card className="max-w-lg mx-auto border-red-500/50">
          <CardHeader>
            <CardTitle className="text-red-400 flex items-center gap-2">
              <span>❌</span>
              {isAuthError ? 'Authentication Error' : isBungieError ? 'Bungie API Error' : 'Failed to Load Collections'}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">{error.message}</p>

            {isAuthError && (
              <div className="bg-yellow-500/10 border border-yellow-500/30 rounded p-3 text-sm">
                <p className="font-medium text-yellow-400">Session may have expired</p>
                <p className="text-muted-foreground mt-1">
                  Try logging out and back in to refresh your Bungie authorization.
                </p>
              </div>
            )}

            {isBungieError && (
              <div className="bg-blue-500/10 border border-blue-500/30 rounded p-3 text-sm">
                <p className="font-medium text-blue-400">Bungie Services Issue</p>
                <p className="text-muted-foreground mt-1">
                  Bungie's servers may be under maintenance. Try again in a few minutes.
                </p>
              </div>
            )}

            <div className="flex gap-2">
              <Button onClick={handleRefresh} variant="outline" size="sm">
                Try Again
              </Button>
              {isAuthError && (
                <Button
                  onClick={() => window.location.href = '/'}
                  variant="default"
                  size="sm"
                >
                  Go to Login
                </Button>
              )}
            </div>

            {/* Debug info for development */}
            {process.env.NODE_ENV === 'development' && (
              <details className="text-xs text-muted-foreground">
                <summary className="cursor-pointer hover:text-foreground">Debug Info</summary>
                <pre className="mt-2 p-2 bg-muted rounded overflow-auto max-h-40">
                  {JSON.stringify(error, null, 2)}
                </pre>
              </details>
            )}
          </CardContent>
        </Card>
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

  const tabs: { key: CollectionTab; label: string; icon: string }[] = [
    { key: 'weapons', label: 'Weapons', icon: '⚔️' },
    { key: 'armor', label: 'Armor', icon: '🛡️' },
    { key: 'exotics', label: 'Exotics', icon: '✨' },
  ];

  return (
    <div className="space-y-6">
      {/* Data Source Banner - for testing */}
      <DataSourceBanner
        collections={collections}
        loading={loading}
        error={error}
        onRefresh={handleRefresh}
      />

      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Collections</h1>
        <p className="text-muted-foreground">
          Track your missing items and plan your acquisitions
        </p>
      </div>

      {/* Collection Tabs */}
      <div className="flex space-x-4 border-b">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
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
                    loading="lazy"
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
                  onClick={() => handleAddToWishList(item)}
                  disabled={addingItems.has(item.itemHash)}
                >
                  {addingItems.has(item.itemHash) ? (
                    <>
                      <LoadingSpinner size="sm" />
                      <span className="ml-2">Adding...</span>
                    </>
                  ) : (
                    'Add to Wish List'
                  )}
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

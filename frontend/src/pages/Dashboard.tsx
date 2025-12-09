import React, { useState } from "react";
import { useQuery } from "@apollo/client";
import { gql } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../components/ui/Card";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { Button } from "../components/ui/Button";
import { useToast } from "../components/ui/Toast";

// Utility functions
const getTimeUntilReset = (): string => {
  const now = new Date();
  const nextTuesday = new Date();
  nextTuesday.setDate(now.getDate() + ((2 - now.getDay() + 7) % 7));
  nextTuesday.setHours(17, 0, 0, 0); // 5 PM UTC

  const diff = nextTuesday.getTime() - now.getTime();
  const days = Math.floor(diff / (1000 * 60 * 60 * 24));
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));

  if (days > 0) {
    return `${days}d ${hours}h`;
  }
  return `${hours}h`;
};

const calculateCollectionProgress = (
  total: number,
  collected: number
): number => {
  if (total === 0) return 0;
  return Math.round((collected / total) * 100);
};

const formatNumber = (num: number): string => {
  return new Intl.NumberFormat().format(num);
};

const CURRENT_USER_QUERY = gql`
  query CurrentUser {
    currentUser {
      id
      displayName
      membershipType
      membershipId
      platform
    }
  }
`;

const USER_COLLECTIONS_QUERY = gql`
  query UserCollections($membershipType: Int!, $membershipId: String!) {
    userCollections(
      membershipType: $membershipType
      membershipId: $membershipId
    ) {
      weapons {
        total
        collected
      }
      armor {
        total
        collected
      }
      exotics {
        total
        collected
      }
    }
  }
`;

// Future use - wishlist functionality
// const WISHLIST_QUERY = gql`
//   query WishList {
//     wishList {
//       id
//       itemHash
//       name
//       priority
//       dateAdded
//     }
//   }
// `;

export function Dashboard() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const { showToast } = useToast();
  const [isRefreshing, setIsRefreshing] = useState(false);

  const { data: userData, refetch: refetchUser } = useQuery(CURRENT_USER_QUERY);
  const { data: collectionsData, loading: collectionsLoading, refetch: refetchCollections } = useQuery(
    USER_COLLECTIONS_QUERY,
    {
      variables: {
        membershipType:
          userData?.currentUser?.membershipType || user?.membershipType,
        membershipId: userData?.currentUser?.membershipId || user?.membershipId,
      },
      skip: !userData?.currentUser && !user?.membershipId,
    }
  );
  // const { data: wishlistData, loading: wishlistLoading } = useQuery(WISHLIST_QUERY); // Future use

  const handleRefreshData = async () => {
    if (isRefreshing) return;

    setIsRefreshing(true);
    try {
      await Promise.all([
        refetchUser(),
        refetchCollections(),
      ]);
      showToast('Data refreshed successfully!', 'success');
    } catch (error) {
      showToast('Failed to refresh data. Please try again.', 'error');
    } finally {
      setIsRefreshing(false);
    }
  };

  if (collectionsLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  const collections = collectionsData?.userCollections;
  // const wishlist = wishlistData?.wishList || []; // Future use

  // Mock weekly data - in a real app this would come from a GraphQL query
  const weekly = {
    vendors: [
      {
        vendorHash: "1",
        vendorName: "Xur",
        location: "Tower Hangar",
        items: [
          {
            itemHash: "1",
            name: "Exotic Engram",
            cost: "97",
            currency: "Legendary Shards",
            icon: "/api/common/destiny2_content/icons/exotic_engram.jpg",
          },
          {
            itemHash: "2",
            name: "Fated Engram",
            cost: "97",
            currency: "Legendary Shards",
            icon: "/api/common/destiny2_content/icons/fated_engram.jpg",
          },
        ],
      },
    ],
    activities: [
      {
        activityHash: "1",
        activityName: "Nightfall: The Ordeal",
        difficulty: "Legend",
        rewards: [
          {
            itemHash: "1",
            name: "Ascendant Shard",
            dropChance: 0.25,
            icon: "/api/common/destiny2_content/icons/ascendant_shard.jpg",
          },
          {
            itemHash: "2",
            name: "Enhancement Prism",
            dropChance: 0.75,
            icon: "/api/common/destiny2_content/icons/enhancement_prism.jpg",
          },
        ],
      },
    ],
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="text-muted-foreground">
            Welcome back, {userData?.currentUser?.displayName}
          </p>
        </div>
        <div className="text-right">
          <p className="text-sm text-muted-foreground">Weekly Reset in</p>
          <p className="text-lg font-semibold">{getTimeUntilReset()}</p>
        </div>
      </div>

      {/* Collection Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="destiny-card-legendary">
          <CardHeader>
            <CardTitle className="text-lg">Weapons</CardTitle>
            <CardDescription>Collection Progress</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Collected</span>
                <span>
                  {collections?.weapons.collected || 0}/
                  {collections?.weapons.total || 0}
                </span>
              </div>
              <div className="w-full bg-muted rounded-full h-2">
                <div
                  className="bg-destiny-legendary h-2 rounded-full"
                  style={{
                    width: `${calculateCollectionProgress(
                      collections?.weapons.total || 0,
                      collections?.weapons.collected || 0
                    )}%`,
                  }}
                />
              </div>
              <p className="text-sm text-muted-foreground">
                {formatNumber(collections?.weapons.missing?.length || 0)}{" "}
                missing items
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="destiny-card-rare">
          <CardHeader>
            <CardTitle className="text-lg">Armor</CardTitle>
            <CardDescription>Collection Progress</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Collected</span>
                <span>
                  {collections?.armor.collected || 0}/
                  {collections?.armor.total || 0}
                </span>
              </div>
              <div className="w-full bg-muted rounded-full h-2">
                <div
                  className="bg-destiny-rare h-2 rounded-full"
                  style={{
                    width: `${calculateCollectionProgress(
                      collections?.armor.total || 0,
                      collections?.armor.collected || 0
                    )}%`,
                  }}
                />
              </div>
              <p className="text-sm text-muted-foreground">
                {formatNumber(collections?.armor.missing?.length || 0)} missing
                items
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="destiny-card-exotic">
          <CardHeader>
            <CardTitle className="text-lg glow-exotic">Exotics</CardTitle>
            <CardDescription>Rare Collection</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Collected</span>
                <span>
                  {collections?.exotics.collected || 0}/
                  {collections?.exotics.total || 0}
                </span>
              </div>
              <div className="w-full bg-muted rounded-full h-2">
                <div
                  className="bg-destiny-exotic h-2 rounded-full"
                  style={{
                    width: `${calculateCollectionProgress(
                      collections?.exotics.total || 0,
                      collections?.exotics.collected || 0
                    )}%`,
                  }}
                />
              </div>
              <p className="text-sm text-muted-foreground">
                {formatNumber(collections?.exotics.missing?.length || 0)}{" "}
                missing exotics
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Weekly Recommendations */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Weekly Vendors</CardTitle>
            <CardDescription>
              Limited-time items available this week
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {weekly?.vendors?.slice(0, 3).map((vendor) => (
              <div key={vendor.vendorHash} className="border rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-semibold">{vendor.vendorName}</h4>
                  <span className="text-sm text-muted-foreground">
                    {vendor.location}
                  </span>
                </div>
                <div className="space-y-2">
                  {vendor.items.slice(0, 2).map((item) => (
                    <div
                      key={item.itemHash}
                      className="flex items-center space-x-3"
                    >
                      <img
                        src={`https://www.bungie.net${item.icon}`}
                        alt={item.name}
                        className="w-8 h-8 rounded"
                      />
                      <div className="flex-1">
                        <p className="text-sm font-medium">{item.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {item.cost} {item.currency}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )) || (
              <p className="text-sm text-muted-foreground">
                No vendor data available
              </p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Featured Activities</CardTitle>
            <CardDescription>
              High-value rewards available this week
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {weekly?.activities?.slice(0, 3).map((activity) => (
              <div
                key={activity.activityHash}
                className="border rounded-lg p-4"
              >
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-semibold">{activity.activityName}</h4>
                  <span className="text-sm text-muted-foreground">
                    {activity.difficulty}
                  </span>
                </div>
                <div className="space-y-2">
                  {activity.rewards.slice(0, 2).map((reward) => (
                    <div
                      key={reward.itemHash}
                      className="flex items-center space-x-3"
                    >
                      <img
                        src={`https://www.bungie.net${reward.icon}`}
                        alt={reward.name}
                        className="w-8 h-8 rounded"
                      />
                      <div className="flex-1">
                        <p className="text-sm font-medium">{reward.name}</p>
                        {reward.dropChance && (
                          <p className="text-xs text-muted-foreground">
                            {Math.round(reward.dropChance * 100)}% drop chance
                          </p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )) || (
              <p className="text-sm text-muted-foreground">
                No activity data available
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <CardTitle>Quick Actions</CardTitle>
          <CardDescription>
            Manage your collection and wish list
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-4">
            <Button onClick={() => navigate("/collections")}>
              View Collections
            </Button>
            <Button variant="outline" onClick={() => navigate("/wishlist")}>
              Manage Wish List
            </Button>
            <Button
              variant="secondary"
              onClick={handleRefreshData}
              disabled={isRefreshing}
            >
              {isRefreshing ? (
                <>
                  <LoadingSpinner size="sm" />
                  <span className="ml-2">Refreshing...</span>
                </>
              ) : (
                'Refresh Data'
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

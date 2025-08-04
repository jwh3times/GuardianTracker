import React, { useState } from "react";
import { useQuery, useMutation } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../components/ui/Card";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { Button } from "../components/ui/Button";
import { GET_WISH_LIST } from "../graphql/queries";
import {
  REMOVE_FROM_WISH_LIST,
  UPDATE_WISH_LIST_ITEM,
} from "../graphql/mutations";
import { WishListItem, WishListPriority } from "../types";
import {
  getRarityColor,
  getPriorityColor,
  getDifficultyColor,
  validateImageUrl,
  formatDate,
} from "../lib/utils";

export function WishList() {
  const navigate = useNavigate();
  const [selectedPriority, setSelectedPriority] = useState<
    WishListPriority | "ALL"
  >("ALL");

  const { data, loading, refetch } = useQuery(GET_WISH_LIST);
  const [removeFromWishList] = useMutation(REMOVE_FROM_WISH_LIST);
  const [updateWishListItem] = useMutation(UPDATE_WISH_LIST_ITEM);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  const wishListItems: WishListItem[] = data?.currentUser?.wishList || [];

  const filteredItems =
    selectedPriority === "ALL"
      ? wishListItems
      : wishListItems.filter((item) => item.priority === selectedPriority);

  const priorityOrder: WishListPriority[] = ["URGENT", "HIGH", "MEDIUM", "LOW"];
  const sortedItems = [...filteredItems].sort((a, b) => {
    const priorityA = priorityOrder.indexOf(a.priority);
    const priorityB = priorityOrder.indexOf(b.priority);
    return priorityA - priorityB;
  });

  const handleRemoveItem = async (itemId: string) => {
    try {
      await removeFromWishList({
        variables: { wishListItemId: itemId },
      });
      await refetch();
    } catch (error) {
      console.error("Error removing item:", error);
    }
  };

  const handleUpdatePriority = async (
    itemId: string,
    priority: WishListPriority
  ) => {
    try {
      await updateWishListItem({
        variables: { wishListItemId: itemId, priority },
      });
      await refetch();
    } catch (error) {
      console.error("Error updating item:", error);
    }
  };

  const priorityStats = priorityOrder.reduce(
    (acc, priority) => {
      acc[priority] = wishListItems.filter(
        (item) => item.priority === priority
      ).length;
      return acc;
    },
    {} as Record<WishListPriority, number>
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Wish List</h1>
        <p className="text-muted-foreground">
          Track and prioritize the items you want to collect
        </p>
      </div>

      {/* Priority Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {priorityOrder.map((priority) => (
          <Card key={priority}>
            <CardContent className="pt-6">
              <div className="text-center">
                <div
                  className={`text-2xl font-bold ${getPriorityColor(priority).split(" ")[0]}`}
                >
                  {priorityStats[priority]}
                </div>
                <p className="text-sm text-muted-foreground capitalize">
                  {priority.toLowerCase()}
                </p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Priority Filter */}
      <Card>
        <CardHeader>
          <CardTitle>Filter by Priority</CardTitle>
          <CardDescription>
            View items by their acquisition priority
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            <Button
              variant={selectedPriority === "ALL" ? "default" : "outline"}
              size="sm"
              onClick={() => setSelectedPriority("ALL")}
            >
              All ({wishListItems.length})
            </Button>
            {priorityOrder.map((priority) => (
              <Button
                key={priority}
                variant={selectedPriority === priority ? "default" : "outline"}
                size="sm"
                onClick={() => setSelectedPriority(priority)}
                className={
                  selectedPriority === priority
                    ? getPriorityColor(priority)
                    : ""
                }
              >
                {priority} ({priorityStats[priority]})
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Wish List Items */}
      <div className="space-y-4">
        {sortedItems.map((item) => (
          <Card
            key={item.id}
            className={`destiny-card ${
              item.isExotic
                ? "destiny-card-exotic"
                : item.rarity === "Legendary"
                  ? "destiny-card-legendary"
                  : "destiny-card-rare"
            }`}
          >
            <CardContent className="p-6">
              <div className="flex items-start space-x-4">
                <img
                  src={validateImageUrl(item.icon)}
                  alt={item.name}
                  className="w-16 h-16 rounded-lg flex-shrink-0"
                />

                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between mb-2">
                    <div>
                      <h3
                        className={`text-lg font-semibold ${getRarityColor(item.rarity)}`}
                      >
                        {item.name}
                      </h3>
                      <p className="text-sm text-muted-foreground">
                        {item.itemType}
                      </p>
                    </div>
                    <div
                      className={`px-3 py-1 rounded-full text-xs font-medium border ${getPriorityColor(item.priority)}`}
                    >
                      {item.priority}
                    </div>
                  </div>

                  <p className="text-sm text-muted-foreground mb-3">
                    {item.description}
                  </p>

                  {item.notes && (
                    <div className="mb-3 p-3 bg-muted rounded-lg">
                      <p className="text-sm font-medium mb-1">Notes:</p>
                      <p className="text-sm text-muted-foreground">
                        {item.notes}
                      </p>
                    </div>
                  )}

                  <div className="flex items-center justify-between text-sm text-muted-foreground mb-4">
                    <span>
                      Difficulty:{" "}
                      <span className={getDifficultyColor(item.difficulty)}>
                        {item.difficulty}
                      </span>
                    </span>
                    <span>Added: {formatDate(item.dateAdded)}</span>
                  </div>

                  {item.sources && item.sources.length > 0 && (
                    <div className="mb-4">
                      <p className="text-sm font-medium mb-2">Sources:</p>
                      <div className="flex flex-wrap gap-2">
                        {item.sources.map((source, index) => (
                          <span
                            key={index}
                            className="px-2 py-1 bg-muted rounded text-xs"
                          >
                            {source}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  <div className="flex items-center space-x-2">
                    <select
                      value={item.priority}
                      onChange={(e) =>
                        handleUpdatePriority(
                          item.id,
                          e.target.value as WishListPriority
                        )
                      }
                      className="px-3 py-1 border rounded text-sm bg-background"
                    >
                      {priorityOrder.map((priority) => (
                        <option key={priority} value={priority}>
                          {priority}
                        </option>
                      ))}
                    </select>

                    <Button variant="outline" size="sm">
                      Edit Notes
                    </Button>

                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleRemoveItem(item.id)}
                    >
                      Remove
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {sortedItems.length === 0 && (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="text-6xl mb-4">📝</div>
            <h3 className="text-lg font-semibold mb-2">
              No Items in Wish List
            </h3>
            <p className="text-muted-foreground mb-4">
              {selectedPriority === "ALL"
                ? "Start building your wish list by adding items from your collections."
                : `No items with ${selectedPriority.toLowerCase()} priority.`}
            </p>
            <Button onClick={() => navigate("/collections")}>
              Browse Collections
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

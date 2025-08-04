import { Context } from "./context";
import { GraphQLError } from "graphql";
import axios from "axios";

const AUTH_SERVICE_URL = "http://localhost:8081";
const BUNGIE_SERVICE_URL = "http://localhost:8082";

export const resolvers = {
  Query: {
    currentUser: async (_parent: any, _args: any, _context: Context) => {
      try {
        console.log("Fetching current user from auth service...");
        const response = await axios.get(
          `${AUTH_SERVICE_URL}/api/auth/profile`
        );
        console.log("Auth service response:", response.data);
        console.log("Returning user data:", response.data.user);
        return response.data.user;
      } catch (error) {
        console.error("Error fetching current user:", error);
        throw new GraphQLError("Failed to fetch user data");
      }
    },

    userCollections: async (
      _parent: any,
      args: { membershipType: number; membershipId: string },
      _context: Context
    ) => {
      try {
        const response = await axios.get(
          `${BUNGIE_SERVICE_URL}/api/collections/${args.membershipType}/${args.membershipId}`
        );
        return response.data;
      } catch (error) {
        console.error("Error fetching user collections:", error);
        throw new GraphQLError("Failed to fetch collections data");
      }
    },

    searchItems: async (
      _parent: any,
      _args: { query: string; filters?: any },
      _context: Context
    ) => {
      try {
        // For now, return empty array since we don't have this endpoint in Go services yet
        // TODO: Implement search endpoint in bungie-service
        return [];
      } catch (error) {
        console.error("Error searching items:", error);
        throw new GraphQLError("Failed to search items");
      }
    },

    weeklyRecommendations: async (
      _parent: any,
      _args: any,
      _context: Context
    ) => {
      try {
        const response = await axios.get(
          `${BUNGIE_SERVICE_URL}/api/weekly/recommendations`
        );
        return response.data;
      } catch (error) {
        console.error("Error fetching weekly recommendations:", error);
        throw new GraphQLError("Failed to fetch weekly recommendations");
      }
    },

    wishList: async (_parent: any, _args: any, _context: Context) => {
      try {
        const response = await axios.get(`${AUTH_SERVICE_URL}/api/wishlist`);
        // Map the Go service response to match GraphQL schema
        const mappedWishList = response.data.map((item: any) => ({
          id: item.id,
          itemHash: item.itemId, // Map itemId to itemHash
          name: item.itemName, // Map itemName to name
          description: `${item.itemType} from Destiny 2`, // Placeholder description
          icon: "", // Placeholder icon
          itemType: item.itemType,
          tierType: 6, // Placeholder tier type (Legendary)
          rarity: "Legendary", // Placeholder rarity
          difficulty: "Moderate", // Placeholder difficulty
          sources: ["Destiny 2"], // Placeholder sources
          isExotic:
            item.itemName.includes("Vex") || item.itemName.includes("Thorn"), // Simple exotic detection
          priority: "MEDIUM", // Default priority
          notes: "", // Default notes
          dateAdded: new Date(item.addedAt), // Convert string to Date object
        }));
        return mappedWishList;
      } catch (error) {
        console.error("Error fetching wish list:", error);
        throw new GraphQLError("Failed to fetch wish list");
      }
    },
  },

  Mutation: {
    login: async (
      _parent: any,
      args: { membershipType: number; membershipId: string },
      _context: Context
    ) => {
      try {
        const loginData = {
          membershipType: args.membershipType,
          membershipId: args.membershipId,
          displayName: `Guardian#${args.membershipId.slice(-4)}`, // Generate display name
        };

        const response = await axios.post(
          `${AUTH_SERVICE_URL}/api/auth/login`,
          loginData
        );

        return {
          token: response.data.token,
          user: response.data.user,
        };
      } catch (error) {
        console.error("Error during login:", error);
        throw new GraphQLError("Failed to authenticate user");
      }
    },

    addToWishList: async (
      _parent: any,
      args: { itemHash: string; priority?: string; notes?: string },
      _context: Context
    ) => {
      try {
        const response = await axios.post(`${AUTH_SERVICE_URL}/api/wishlist`, {
          itemId: args.itemHash,
          priority: args.priority || "MEDIUM",
          notes: args.notes,
        });
        return response.data;
      } catch (error) {
        console.error("Error adding to wish list:", error);
        throw new GraphQLError("Failed to add item to wish list");
      }
    },

    removeFromWishList: async (
      _parent: any,
      args: { wishListItemId: string },
      _context: Context
    ) => {
      try {
        const response = await axios.delete(
          `${AUTH_SERVICE_URL}/api/wishlist/${args.wishListItemId}`
        );
        return { success: true, message: response.data.message };
      } catch (error) {
        console.error("Error removing from wish list:", error);
        throw new GraphQLError("Failed to remove item from wish list");
      }
    },

    updateWishListItem: async (
      _parent: any,
      args: { wishListItemId: string; priority?: string; notes?: string },
      _context: Context
    ) => {
      try {
        // TODO: Implement PUT endpoint in auth-service for updating wishlist items
        // For now, return success message
        return {
          id: args.wishListItemId,
          priority: args.priority || "MEDIUM",
          notes: args.notes || "",
          message: "Wishlist item updated successfully",
        };
      } catch (error) {
        console.error("Error updating wish list item:", error);
        throw new GraphQLError("Failed to update wish list item");
      }
    },

    refreshUserData: async (_parent: any, _args: any, context: Context) => {
      if (!context.user) {
        throw new GraphQLError("Authentication required");
      }

      try {
        // TODO: Implement refresh endpoint in bungie-service
        // For now, return success message
        return {
          success: true,
          message: "User data refreshed successfully",
          lastUpdated: new Date().toISOString(),
        };
      } catch (error) {
        console.error("Error refreshing user data:", error);
        throw new GraphQLError("Failed to refresh user data");
      }
    },

    logout: async (_parent: any, _args: any, context: Context) => {
      if (!context.user) {
        throw new GraphQLError("Authentication required");
      }

      try {
        // TODO: Implement logout endpoint in auth-service
        // For now, return success message
        return {
          success: true,
          message: "Successfully logged out",
        };
      } catch (error) {
        console.error("Error during logout:", error);
        throw new GraphQLError("Failed to logout");
      }
    },
  },

  // Custom scalar resolvers
  DateTime: {
    serialize: (date: Date) => date.toISOString(),
    parseValue: (value: string) => new Date(value),
    parseLiteral: (ast: any) => new Date(ast.value),
  },
};

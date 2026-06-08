import { Context } from "./context";
import { GraphQLError } from "graphql";
import axios from "axios";
import { requireAuth, getAuthenticatedUser } from "./utils/auth";
import { validateInput, validationSchemas } from "./utils/validation";

const AUTH_SERVICE_URL =
  process.env.AUTH_SERVICE_URL || "http://localhost:8081";
const BUNGIE_SERVICE_URL =
  process.env.BUNGIE_SERVICE_URL || "http://localhost:8082";

// Only log in non-production environments
if (process.env.NODE_ENV !== "production") {
  console.log("GraphQL Resolvers Configuration:");
  console.log("  AUTH_SERVICE_URL:", AUTH_SERVICE_URL);
  console.log("  BUNGIE_SERVICE_URL:", BUNGIE_SERVICE_URL);
}

// Helper to forward auth token to downstream services
function getAuthHeaders(context: Context): Record<string, string> {
  const authHeader = context.req.headers.authorization;
  return authHeader ? { Authorization: authHeader } : {};
}

export const resolvers = {
  Query: {
    // Protected: Requires authentication
    currentUser: async (_parent: unknown, _args: unknown, context: Context) => {
      requireAuth(context);

      try {
        const response = await axios.get(
          `${AUTH_SERVICE_URL}/api/auth/profile`,
          { headers: getAuthHeaders(context) }
        );
        return response.data.user;
      } catch (error) {
        console.error("Error fetching current user:", error);
        throw new GraphQLError("Failed to fetch user data");
      }
    },

    // Protected: Requires authentication + validates input
    characters: async (
      _parent: unknown,
      args: { membershipType: number; membershipId: string },
      context: Context
    ) => {
      requireAuth(context);
      const validated = validateInput(validationSchemas.characters, args);

      try {
        const response = await axios.get(
          `${BUNGIE_SERVICE_URL}/api/characters/${validated.membershipType}/${validated.membershipId}`,
          { headers: getAuthHeaders(context) }
        );

        // Coerce the ISO timestamp into a Date so the DateTime scalar serializes it.
        return (response.data as Record<string, unknown>[]).map((ch) => ({
          ...ch,
          dateLastPlayed: ch.dateLastPlayed
            ? new Date(ch.dateLastPlayed as string)
            : null,
        }));
      } catch (error) {
        console.error("Error fetching characters:", error);
        throw new GraphQLError("Failed to fetch characters");
      }
    },

    // Protected: Requires authentication + validates input
    userCollections: async (
      _parent: unknown,
      args: { membershipType: number; membershipId: string },
      context: Context
    ) => {
      requireAuth(context);
      const validated = validateInput(validationSchemas.userCollections, args);

      try {
        const response = await axios.get(
          `${BUNGIE_SERVICE_URL}/api/collections/${validated.membershipType}/${validated.membershipId}`,
          { headers: getAuthHeaders(context) }
        );
        return response.data;
      } catch (error) {
        console.error("Error fetching user collections:", error);
        throw new GraphQLError("Failed to fetch collections data");
      }
    },

    // Protected: Requires authentication + validates input
    searchItems: async (
      _parent: unknown,
      args: { query: string; filters?: Record<string, unknown> },
      context: Context
    ) => {
      requireAuth(context);
      const validated = validateInput(validationSchemas.searchItems, args);

      try {
        // TODO: Implement search endpoint in bungie-service
        console.log("Searching items with query:", validated.query);
        return [];
      } catch (error) {
        console.error("Error searching items:", error);
        throw new GraphQLError("Failed to search items");
      }
    },

    // Public: Weekly recommendations (no auth required)
    weeklyRecommendations: async (
      _parent: unknown,
      _args: unknown,
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

    // Protected: Requires authentication
    wishList: async (_parent: unknown, _args: unknown, context: Context) => {
      requireAuth(context);

      try {
        const response = await axios.get(`${AUTH_SERVICE_URL}/api/wishlist`, {
          headers: getAuthHeaders(context),
        });

        // Map the Go service response to match GraphQL schema
        const mappedWishList = response.data.map((item: Record<string, unknown>) => ({
          id: item.id,
          itemHash: item.itemId,
          name: item.itemName,
          description: `${item.itemType} from Destiny 2`,
          icon: "",
          itemType: item.itemType,
          tierType: 6,
          rarity: "Legendary",
          difficulty: "Moderate",
          sources: ["Destiny 2"],
          isExotic:
            String(item.itemName).includes("Vex") ||
            String(item.itemName).includes("Thorn"),
          priority: "MEDIUM",
          notes: "",
          dateAdded: new Date(item.addedAt as string),
        }));
        return mappedWishList;
      } catch (error) {
        console.error("Error fetching wish list:", error);
        throw new GraphQLError("Failed to fetch wish list");
      }
    },
  },

  Mutation: {
    // Login: Validates input (creates session)
    login: async (
      _parent: unknown,
      args: { membershipType: number; membershipId: string },
      _context: Context
    ) => {
      const validated = validateInput(validationSchemas.login, args);

      try {
        const loginData = {
          membershipType: validated.membershipType,
          membershipId: validated.membershipId,
          displayName: `Guardian#${validated.membershipId.slice(-4)}`,
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

    // Protected: Requires authentication + validates input
    addToWishList: async (
      _parent: unknown,
      args: { itemHash: string; priority?: string; notes?: string },
      context: Context
    ) => {
      requireAuth(context);
      const validated = validateInput(validationSchemas.addToWishList, args);

      try {
        const response = await axios.post(
          `${AUTH_SERVICE_URL}/api/wishlist`,
          {
            itemId: validated.itemHash,
            priority: validated.priority || "MEDIUM",
            notes: validated.notes,
          },
          { headers: getAuthHeaders(context) }
        );
        return response.data;
      } catch (error) {
        console.error("Error adding to wish list:", error);
        throw new GraphQLError("Failed to add item to wish list");
      }
    },

    // Protected: Requires authentication + validates input
    removeFromWishList: async (
      _parent: unknown,
      args: { wishListItemId: string },
      context: Context
    ) => {
      requireAuth(context);
      const validated = validateInput(validationSchemas.removeFromWishList, args);

      try {
        const response = await axios.delete(
          `${AUTH_SERVICE_URL}/api/wishlist/${validated.wishListItemId}`,
          { headers: getAuthHeaders(context) }
        );
        return { success: true, message: response.data.message };
      } catch (error) {
        console.error("Error removing from wish list:", error);
        throw new GraphQLError("Failed to remove item from wish list");
      }
    },

    // Protected: Requires authentication + validates input
    updateWishListItem: async (
      _parent: unknown,
      args: { wishListItemId: string; priority?: string; notes?: string },
      context: Context
    ) => {
      requireAuth(context);
      const validated = validateInput(validationSchemas.updateWishListItem, args);

      try {
        // TODO: Implement PUT endpoint in auth-service for updating wishlist items
        return {
          id: validated.wishListItemId,
          priority: validated.priority || "MEDIUM",
          notes: validated.notes || "",
          message: "Wishlist item updated successfully",
        };
      } catch (error) {
        console.error("Error updating wish list item:", error);
        throw new GraphQLError("Failed to update wish list item");
      }
    },

    // Protected: Requires authentication
    refreshUserData: async (_parent: unknown, _args: unknown, context: Context) => {
      getAuthenticatedUser(context); // Validates auth, throws if not authenticated

      try {
        // TODO: Implement refresh endpoint in bungie-service
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

    // Protected: Requires authentication
    logout: async (_parent: unknown, _args: unknown, context: Context) => {
      getAuthenticatedUser(context); // Validates auth, throws if not authenticated

      try {
        // TODO: Implement logout endpoint in auth-service (token blacklist)
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
    parseLiteral: (ast: { value: string }) => new Date(ast.value),
  },
};

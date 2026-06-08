import { GraphQLError } from "graphql";
import { Context } from "../context.js";

/**
 * Authentication guard - throws error if user is not authenticated
 */
export function requireAuth(context: Context): void {
  if (!context.user) {
    throw new GraphQLError("Authentication required", {
      extensions: {
        code: "UNAUTHENTICATED",
        http: { status: 401 },
      },
    });
  }
}

/**
 * Get authenticated user or throw
 */
export function getAuthenticatedUser(context: Context): NonNullable<Context["user"]> {
  requireAuth(context);
  return context.user!;
}

/**
 * Check if user owns a resource
 */
export function requireOwnership(
  context: Context,
  resourceOwnerId: string
): void {
  const user = getAuthenticatedUser(context);
  if (user.membershipId !== resourceOwnerId && user.membership_id !== resourceOwnerId) {
    throw new GraphQLError("You don't have permission to access this resource", {
      extensions: {
        code: "FORBIDDEN",
        http: { status: 403 },
      },
    });
  }
}

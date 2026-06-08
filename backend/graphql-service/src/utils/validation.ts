import { z } from "zod";
import { GraphQLError } from "graphql";

// Bungie membership types (1=Xbox, 2=PSN, 3=Steam, etc.)
const VALID_MEMBERSHIP_TYPES = [1, 2, 3, 4, 5, 6, 10, 254] as const;

// Common validation schemas
export const schemas = {
  // Membership ID - numeric string, 15-20 characters
  membershipId: z
    .string()
    .min(10, "Membership ID must be at least 10 characters")
    .max(25, "Membership ID must be at most 25 characters")
    .regex(/^\d+$/, "Membership ID must contain only digits"),

  // Membership type - valid Bungie platform
  membershipType: z
    .number()
    .int("Membership type must be an integer")
    .refine(
      (val) => VALID_MEMBERSHIP_TYPES.includes(val as any),
      "Invalid membership type"
    ),

  // Item hash - Bungie item hash (numeric string or number)
  itemHash: z
    .string()
    .min(1, "Item hash is required")
    .max(20, "Item hash too long")
    .regex(/^\d+$/, "Item hash must be numeric"),

  // Wishlist item ID
  wishListItemId: z
    .string()
    .min(1, "Wishlist item ID is required")
    .max(50, "Wishlist item ID too long"),

  // Priority enum
  priority: z.enum(["LOW", "MEDIUM", "HIGH"]).optional(),

  // Notes - sanitized text
  notes: z
    .string()
    .max(500, "Notes must be 500 characters or less")
    .optional()
    .transform((val) => val?.trim()),

  // Search query
  searchQuery: z
    .string()
    .min(2, "Search query must be at least 2 characters")
    .max(100, "Search query must be at most 100 characters")
    .transform((val) => val.trim()),
};

// Composite schemas for resolver arguments
export const validationSchemas = {
  userCollections: z.object({
    membershipType: schemas.membershipType,
    membershipId: schemas.membershipId,
  }),

  characters: z.object({
    membershipType: schemas.membershipType,
    membershipId: schemas.membershipId,
  }),

  login: z.object({
    membershipType: schemas.membershipType,
    membershipId: schemas.membershipId,
  }),

  addToWishList: z.object({
    itemHash: schemas.itemHash,
    priority: schemas.priority,
    notes: schemas.notes,
  }),

  removeFromWishList: z.object({
    wishListItemId: schemas.wishListItemId,
  }),

  updateWishListItem: z.object({
    wishListItemId: schemas.wishListItemId,
    priority: schemas.priority,
    notes: schemas.notes,
  }),

  searchItems: z.object({
    query: schemas.searchQuery,
    filters: z
      .object({
        itemType: z.string().max(50).optional(),
        tierType: z.number().int().min(0).max(6).optional(),
        isExotic: z.boolean().optional(),
      })
      .optional(),
  }),
};

/**
 * Validate input against a schema, throw GraphQL error on failure
 */
export function validateInput<T>(
  schema: z.ZodSchema<T>,
  input: unknown
): T {
  const result = schema.safeParse(input);

  if (!result.success) {
    const errors = result.error.issues.map((issue) => ({
      field: issue.path.join("."),
      message: issue.message,
    }));

    throw new GraphQLError("Validation failed", {
      extensions: {
        code: "BAD_USER_INPUT",
        validationErrors: errors,
        http: { status: 400 },
      },
    });
  }

  return result.data;
}

import { gql } from "graphql-tag";

export const typeDefs = gql`
  # Scalar types
  scalar DateTime

  # Enums
  enum WishListPriority {
    LOW
    MEDIUM
    HIGH
    URGENT
  }

  enum ItemDifficulty {
    Easy
    Moderate
    Challenging
  }

  enum ItemRarity {
    Common
    Uncommon
    Rare
    Legendary
    Exotic
  }

  # Core Types
  type User {
    id: ID!
    displayName: String!
    membershipType: Int!
    membershipId: String!
    platform: String
    profilePicture: String
    lastLoginAt: DateTime
    createdAt: DateTime
    wishList: [WishListItem!]!
  }

  type Character {
    characterId: ID!
    classType: Int!
    className: String!
    raceName: String!
    light: Int!
    emblemPath: String!
    emblemBackgroundPath: String!
    dateLastPlayed: DateTime
  }

  type DestinyItem {
    itemHash: String!
    name: String!
    description: String!
    icon: String!
    itemType: String!
    tierType: Int!
    rarity: ItemRarity!
    difficulty: ItemDifficulty!
    sources: [String!]!
    isExotic: Boolean!
    isCollected: Boolean
  }

  type WishListItem {
    id: ID!
    itemHash: String!
    name: String!
    description: String!
    icon: String!
    itemType: String!
    tierType: Int!
    rarity: ItemRarity!
    difficulty: ItemDifficulty!
    sources: [String!]!
    isExotic: Boolean!
    priority: WishListPriority!
    notes: String
    dateAdded: DateTime!
  }

  type CollectionSummary {
    total: Int!
    collected: Int!
    missing: [DestinyItem!]!
  }

  type UserCollections {
    weapons: CollectionSummary!
    armor: CollectionSummary!
    exotics: CollectionSummary!
  }

  # Weekly Content Types
  type VendorItem {
    itemHash: String!
    name: String!
    description: String!
    icon: String!
    cost: Int!
    currency: String!
    isLimited: Boolean!
    endsAt: DateTime
  }

  type Vendor {
    vendorHash: String!
    vendorName: String!
    location: String!
    items: [VendorItem!]!
  }

  type ActivityReward {
    itemHash: String!
    name: String!
    description: String!
    icon: String!
    dropChance: Float
  }

  type Activity {
    activityHash: String!
    activityName: String!
    activityType: String!
    difficulty: String!
    rewards: [ActivityReward!]!
    isWeeklyFeatured: Boolean!
    endsAt: DateTime
  }

  type PursuitObjective {
    description: String!
    progress: Int!
    completionValue: Int!
    isCompleted: Boolean!
  }

  type Pursuit {
    itemHash: String!
    name: String!
    description: String!
    icon: String!
    objectives: [PursuitObjective!]!
    rewards: [DestinyItem!]!
    expirationDate: DateTime
  }

  type WeeklyRecommendations {
    vendors: [Vendor!]!
    activities: [Activity!]!
    pursuits: [Pursuit!]!
  }

  # Input Types
  input ItemFilters {
    itemType: [String!]
    rarity: [ItemRarity!]
    difficulty: [ItemDifficulty!]
    isExotic: Boolean
    isCollected: Boolean
    sources: [String!]
  }

  # Response Types
  type MutationResponse {
    success: Boolean!
    message: String!
  }

  type AuthResponse {
    token: String!
    user: User!
  }

  type RefreshDataResponse {
    success: Boolean!
    message: String!
    lastRefresh: DateTime!
  }

  # Queries
  type Query {
    # User queries
    currentUser: User

    # Characters
    characters(membershipType: Int!, membershipId: String!): [Character!]!

    # Collection queries
    userCollections(
      membershipType: Int!
      membershipId: String!
    ): UserCollections
    searchItems(query: String!, filters: ItemFilters): [DestinyItem!]!

    # Weekly content
    weeklyRecommendations: WeeklyRecommendations

    # Wish list
    wishList: [WishListItem!]!
  }

  # Mutations
  type Mutation {
    # Authentication
    login(membershipType: Int!, membershipId: String!): AuthResponse!

    # Wish list mutations
    addToWishList(
      itemHash: String!
      priority: WishListPriority
      notes: String
    ): WishListItem!
    removeFromWishList(wishListItemId: ID!): MutationResponse!
    updateWishListItem(
      wishListItemId: ID!
      priority: WishListPriority
      notes: String
    ): WishListItem!

    # Data management
    refreshUserData: RefreshDataResponse!

    # Authentication
    logout: MutationResponse!
  }
`;

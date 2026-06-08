import { gql, type TypedDocumentNode } from '@apollo/client';
import type {
  DestinyItem,
  ItemFilters,
  User,
  UserCollections,
  WeeklyRecommendations,
  WishListItem,
} from '../types';
import type { GraphQLCharacter } from '../lib/adapters';

type MembershipVars = { membershipType: number; membershipId: string };

export const GET_CURRENT_USER: TypedDocumentNode<{ currentUser: User | null }> = gql`
  query GetCurrentUser {
    currentUser {
      id
      displayName
      membershipType
      membershipId
      profilePicture
      lastLoginAt
      createdAt
    }
  }
`;

export const GET_CHARACTERS: TypedDocumentNode<
  { characters: GraphQLCharacter[] },
  MembershipVars
> = gql`
  query GetCharacters($membershipType: Int!, $membershipId: String!) {
    characters(membershipType: $membershipType, membershipId: $membershipId) {
      characterId
      classType
      className
      raceName
      light
      emblemPath
      emblemBackgroundPath
      dateLastPlayed
    }
  }
`;

export const GET_USER_COLLECTIONS: TypedDocumentNode<
  { userCollections: UserCollections | null },
  MembershipVars
> = gql`
  query GetUserCollections($membershipType: Int!, $membershipId: String!) {
    userCollections(membershipType: $membershipType, membershipId: $membershipId) {
      weapons {
        total
        collected
        missing {
          itemHash
          name
          description
          icon
          itemType
          tierType
          rarity
          difficulty
          sources
          isExotic
        }
      }
      armor {
        total
        collected
        missing {
          itemHash
          name
          description
          icon
          itemType
          tierType
          rarity
          difficulty
          sources
          isExotic
        }
      }
      exotics {
        total
        collected
        missing {
          itemHash
          name
          description
          icon
          itemType
          tierType
          rarity
          difficulty
          sources
          isExotic
        }
      }
    }
  }
`;

export const GET_WISH_LIST: TypedDocumentNode<{
  currentUser: { id: string; wishList: WishListItem[] } | null;
}> = gql`
  query GetWishList {
    currentUser {
      id
      wishList {
        id
        itemHash
        name
        description
        icon
        itemType
        tierType
        rarity
        difficulty
        sources
        isExotic
        priority
        notes
        dateAdded
      }
    }
  }
`;

export const GET_WEEKLY_RECOMMENDATIONS: TypedDocumentNode<{
  weeklyRecommendations: WeeklyRecommendations | null;
}> = gql`
  query GetWeeklyRecommendations {
    weeklyRecommendations {
      vendors {
        vendorHash
        vendorName
        location
        items {
          itemHash
          name
          description
          icon
          cost
          currency
          isLimited
          endsAt
        }
      }
      activities {
        activityHash
        activityName
        activityType
        difficulty
        rewards {
          itemHash
          name
          description
          icon
          dropChance
        }
        isWeeklyFeatured
        endsAt
      }
      pursuits {
        itemHash
        name
        description
        icon
        objectives {
          description
          progress
          completionValue
          isCompleted
        }
        rewards {
          itemHash
          name
          description
          icon
        }
        expirationDate
      }
    }
  }
`;

export const SEARCH_ITEMS: TypedDocumentNode<
  { searchItems: DestinyItem[] },
  { query: string; filters?: ItemFilters }
> = gql`
  query SearchItems($query: String!, $filters: ItemFilters) {
    searchItems(query: $query, filters: $filters) {
      itemHash
      name
      description
      icon
      itemType
      tierType
      rarity
      difficulty
      sources
      isExotic
      isCollected
    }
  }
`;

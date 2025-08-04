import { gql } from '@apollo/client';

export const ADD_TO_WISH_LIST = gql`
  mutation AddToWishList($itemHash: String!, $priority: WishListPriority, $notes: String) {
    addToWishList(itemHash: $itemHash, priority: $priority, notes: $notes) {
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
`;

export const REMOVE_FROM_WISH_LIST = gql`
  mutation RemoveFromWishList($wishListItemId: ID!) {
    removeFromWishList(wishListItemId: $wishListItemId) {
      success
      message
    }
  }
`;

export const UPDATE_WISH_LIST_ITEM = gql`
  mutation UpdateWishListItem($wishListItemId: ID!, $priority: WishListPriority, $notes: String) {
    updateWishListItem(wishListItemId: $wishListItemId, priority: $priority, notes: $notes) {
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
`;

export const REFRESH_USER_DATA = gql`
  mutation RefreshUserData {
    refreshUserData {
      success
      message
      lastRefresh
    }
  }
`;

export const LOGOUT = gql`
  mutation Logout {
    logout {
      success
      message
    }
  }
`;

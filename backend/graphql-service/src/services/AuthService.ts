import axios from 'axios';

export class AuthService {
  private baseURL: string;

  constructor() {
    this.baseURL = process.env.AUTH_SERVICE_URL || 'http://localhost:8081';
  }

  async getCurrentUser(userId: string) {
    try {
      const response = await axios.get(`${this.baseURL}/api/users/${userId}`);
      return response.data;
    } catch (error) {
      console.error('Error fetching user:', error);
      throw error;
    }
  }

  async addToWishList(userId: string, itemHash: string, priority: string, notes?: string) {
    try {
      const response = await axios.post(`${this.baseURL}/api/users/${userId}/wishlist`, {
        itemHash,
        priority,
        notes,
      });
      return response.data;
    } catch (error) {
      console.error('Error adding to wish list:', error);
      throw error;
    }
  }

  async removeFromWishList(userId: string, wishListItemId: string) {
    try {
      await axios.delete(`${this.baseURL}/api/users/${userId}/wishlist/${wishListItemId}`);
    } catch (error) {
      console.error('Error removing from wish list:', error);
      throw error;
    }
  }

  async updateWishListItem(userId: string, wishListItemId: string, priority?: string, notes?: string) {
    try {
      const response = await axios.put(`${this.baseURL}/api/users/${userId}/wishlist/${wishListItemId}`, {
        priority,
        notes,
      });
      return response.data;
    } catch (error) {
      console.error('Error updating wish list item:', error);
      throw error;
    }
  }

  async logout(userId: string) {
    try {
      await axios.post(`${this.baseURL}/api/auth/logout`, { userId });
    } catch (error) {
      console.error('Error during logout:', error);
      throw error;
    }
  }
}

import axios from 'axios';

export class BungieService {
  private baseURL: string;

  constructor() {
    this.baseURL = process.env.BUNGIE_SERVICE_URL || 'http://localhost:8082';
  }

  async getUserCollections(membershipType: number, membershipId: string) {
    try {
      const response = await axios.get(`${this.baseURL}/api/collections/${membershipType}/${membershipId}`);
      return response.data;
    } catch (error) {
      console.error('Error fetching user collections:', error);
      throw error;
    }
  }

  async searchItems(query: string, filters?: any) {
    try {
      const response = await axios.get(`${this.baseURL}/api/items/search`, {
        params: { query, ...filters },
      });
      return response.data;
    } catch (error) {
      console.error('Error searching items:', error);
      throw error;
    }
  }

  async getWeeklyRecommendations() {
    try {
      const response = await axios.get(`${this.baseURL}/api/weekly/recommendations`);
      return response.data;
    } catch (error) {
      console.error('Error fetching weekly recommendations:', error);
      throw error;
    }
  }

  async refreshUserData(membershipType: number, membershipId: string) {
    try {
      const response = await axios.post(`${this.baseURL}/api/collections/${membershipType}/${membershipId}/refresh`);
      return response.data;
    } catch (error) {
      console.error('Error refreshing user data:', error);
      throw error;
    }
  }

  async getItemDetails(itemHash: string) {
    try {
      const response = await axios.get(`${this.baseURL}/api/items/${itemHash}`);
      return response.data;
    } catch (error) {
      console.error('Error fetching item details:', error);
      throw error;
    }
  }
}

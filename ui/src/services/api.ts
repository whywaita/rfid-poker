import { CardType } from '@/components/Cards';

export type ApiPlayerType = {
  id: number;
  name: string;
  device_id: string;
  pair_id: number;
};

export type ApiAntennaType = {
  id: number;
  device_id: string;
  pair_id: number;
  antenna_type_name: string;
};

export type ApiHandType = {
  id: number;
  player_id: number;
  cards: CardType[];
  is_muck: boolean;
};

export class ApiService {
    constructor(private baseUrl: string) {}
  
    private async handleResponse<T>(response: Response): Promise<T> {
      if (response.status === 404) {
        throw new Error('Not found');
      }
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
      }
      return response.json();
    }

    async getPlayers(): Promise<{players: ApiPlayerType[]}> {
      const response = await fetch(`${this.baseUrl}/admin/player`);
      return this.handleResponse(response);
    }

    async getAntennas(): Promise<{antenna: ApiAntennaType[]}> {
      const response = await fetch(`${this.baseUrl}/admin/antenna`);
      return this.handleResponse(response);
    }

    async getPlayerHand(playerId: number): Promise<{hand: ApiHandType}> {
      const response = await fetch(`${this.baseUrl}/admin/player/${playerId}/hand`);
      return this.handleResponse(response);
    }
  
    async updatePlayer(playerId: number, name: string): Promise<void> {
      const response = await fetch(`${this.baseUrl}/admin/player/${playerId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      });
      return this.handleResponse(response);
    }

    async updateAntennaType(antennaId: number, antenna_type_name: string): Promise<void> {
      const response = await fetch(`${this.baseUrl}/admin/antenna/${antennaId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ antenna_type_name })
      });
      return this.handleResponse(response);
    }

    async deleteAntenna(antennaId: number): Promise<void> {
      const response = await fetch(`${this.baseUrl}/admin/antenna/${antennaId}`, {
        method: 'DELETE'
      });
      return this.handleResponse(response);
    }

    async muckHand(playerId: number): Promise<void> {
      const response = await fetch(`${this.baseUrl}/admin/player/${playerId}/hand/muck`, {
        method: 'POST'
      });
      return this.handleResponse(response);
    }

    async resetGame(): Promise<void> {
      const response = await fetch(`${this.baseUrl}/admin/game`, {
        method: 'DELETE'
      });
      return this.handleResponse(response);
    }
} 
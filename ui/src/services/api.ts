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
    private readonly httpUrl: string;

    constructor(baseUrl: string) {
        this.httpUrl = ApiService.convertToHttpUrl(baseUrl);
    }

    private static convertToHttpUrl(wsUrl: string): string {
        if (wsUrl.startsWith('wss://')) {
            return wsUrl.replace('wss://', 'https://');
        } else if (wsUrl.startsWith('ws://')) {
            return wsUrl.replace('ws://', 'http://');
        }
        return wsUrl;
    }
  
    private async handleResponse<T>(response: Response): Promise<T> {
        if (response.status === 204) {
            return {} as T;
        }
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
        const response = await fetch(`${this.httpUrl}/admin/player`);
        return this.handleResponse(response);
    }

    async getAntennas(): Promise<{antenna: ApiAntennaType[]}> {
      const response = await fetch(`${this.httpUrl}/admin/antenna`);
      return this.handleResponse(response);
    }

    async getPlayerHand(playerId: number): Promise<{hand: ApiHandType}> {
      const response = await fetch(`${this.httpUrl}/admin/player/${playerId}/hand`);
      return this.handleResponse(response);
    }
  
    async updatePlayer(playerId: number, name: string): Promise<void> {
      const response = await fetch(`${this.httpUrl}/admin/player/${playerId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      });
      return this.handleResponse(response);
    }

    async updateAntennaType(antennaId: number, antenna_type_name: string): Promise<void> {
      const response = await fetch(`${this.httpUrl}/admin/antenna/${antennaId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ antenna_type_name })
      });
      return this.handleResponse(response);
    }

    async deleteAntenna(antennaId: number): Promise<void> {
      const response = await fetch(`${this.httpUrl}/admin/antenna/${antennaId}`, {
        method: 'DELETE'
      });
      return this.handleResponse(response);
    }

    async muckHand(playerId: number): Promise<void> {
      const response = await fetch(`${this.httpUrl}/admin/player/${playerId}/hand`, {
        method: 'DELETE'
      });
      return this.handleResponse(response);
    }

    async resetGame(): Promise<void> {
      const response = await fetch(`${this.httpUrl}/admin/game`, {
        method: 'DELETE'
      });
      return this.handleResponse(response);
    }
} 
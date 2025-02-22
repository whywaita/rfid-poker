import { useState, useEffect } from 'react';
import ConfirmationModal from '@/components/ConfirmationModal';
import ConnectionModal from '@/components/ConnectionModal';
import Card, { CardType } from '@/components/Cards';
import { ApiService, ApiPlayerType, ApiAntennaType } from '@/services/api';

const ANTENNA_TYPES = ['player', 'muck', 'board', 'unknown'] as const;
type AntennaTypeName = typeof ANTENNA_TYPES[number];

export default function Admin() {
  const [modalOpen, setModalOpen] = useState(true);
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);
  const [confirmAntennaDeleteModal, setConfirmAntennaDeleteModal] = useState<{open: boolean, antennaId: number | null}>({
    open: false,
    antennaId: null
  });
  const [hostname, setHostname] = useState("");
  const [players, setPlayers] = useState<ApiPlayerType[]>([]);
  const [antennas, setAntennas] = useState<ApiAntennaType[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [playerHands, setPlayerHands] = useState<{[key: number]: {cards: CardType[], is_muck: boolean, error?: string}}>({});
  const [api, setApi] = useState<ApiService | null>(null);

  useEffect(() => {
    const storedHostname = localStorage.getItem("hostname");
    if (storedHostname) {
      handleHostnameSubmit(storedHostname);
    }
  }, []);

  useEffect(() => {
    if (api && players.length > 0) {
      players.forEach(player => {
        fetchPlayerHand(player.id);
      });
    }
  }, [api, players]);

  const handleHostnameSubmit = (inputHostname: string) => {
    const httpUrl = getHttpUrl(inputHostname);
    setHostname(inputHostname);
    localStorage.setItem("hostname", inputHostname);
    const apiService = new ApiService(httpUrl);
    setApi(apiService);
    setModalOpen(false);
    fetchData(apiService);
  };

  const fetchData = async (apiService: ApiService) => {
    try {
      const [playersResponse, antennasResponse] = await Promise.all([
        apiService.getPlayers(),
        apiService.getAntennas()
      ]);
      setPlayers(playersResponse.players);
      setAntennas(antennasResponse.antenna);
      setError(null);
    } catch (error) {
      console.error('Failed to fetch data:', error);
      setError(error instanceof Error ? error.message : 'Failed to fetch data');
    }
  };

  const fetchPlayerHand = async (playerId: number) => {
    if (!api) return;
    try {
      const response = await api.getPlayerHand(playerId);
      setPlayerHands(prev => ({
        ...prev,
        [playerId]: {
          cards: response.hand.cards,
          is_muck: response.hand.is_muck
        }
      }));
    } catch (error) {
      if (error instanceof Error && error.message === 'Not found') {
        setPlayerHands(prev => ({
          ...prev,
          [playerId]: {
            cards: [],
            is_muck: false,
            error: 'Hand not found (404)'
          }
        }));
        return;
      }
      console.error(`Failed to fetch hand for player ${playerId}:`, error);
    }
  };

  function removeHostname() {
    localStorage.removeItem("hostname");
    setHostname("");
    setApi(null);
    setModalOpen(true);
  }

  const getHttpUrl = (wsUrl: string) => {
    if (wsUrl.startsWith('wss://')) {
      return wsUrl.replace('wss://', 'https://');
    } else if (wsUrl.startsWith('ws://')) {
      return wsUrl.replace('ws://', 'http://');
    }
    return wsUrl;
  };

  const handlePlayerSubmit = async (player: ApiPlayerType, newName: string) => {
    if (!api) return;
    try {
      await api.updatePlayer(player.id, newName);
      const { players } = await api.getPlayers();
      setPlayers(players);
      setError(null);
    } catch (error) {
      console.error('Failed to update player name:', error);
      setError(error instanceof Error ? error.message : 'Failed to update player name');
    }
  };

  const handleAntennaTypeSubmit = async (antenna: ApiAntennaType, newType: AntennaTypeName) => {
    if (!api) return;
    try {
      await api.updateAntennaType(antenna.id, newType);
      const { antenna: updatedAntennas } = await api.getAntennas();
      setAntennas(updatedAntennas);
      setError(null);
    } catch (error) {
      console.error('Failed to update antenna type:', error);
      setError(error instanceof Error ? error.message : 'Failed to update antenna type');
    }
  };

  const handleDeleteAntenna = async (antennaId: number) => {
    if (!api) return;
    try {
      await api.deleteAntenna(antennaId);
      const { antenna } = await api.getAntennas();
      setAntennas(antenna);
      setError(null);
      setConfirmAntennaDeleteModal({open: false, antennaId: null});
    } catch (error) {
      console.error('Failed to delete antenna:', error);
      setError(error instanceof Error ? error.message : 'Failed to delete antenna');
      setConfirmAntennaDeleteModal({open: false, antennaId: null});
    }
  };

  const handleMuckHand = async (playerId: number) => {
    if (!api) return;
    try {
      await api.muckHand(playerId);
      await fetchPlayerHand(playerId);
      setError(null);
    } catch (error) {
      console.error('Failed to muck hand:', error);
      setError(error instanceof Error ? error.message : 'Failed to muck hand');
    }
  };

  const handleResetGame = async () => {
    if (!api) return;
    try {
      await api.resetGame();
      const { players } = await api.getPlayers();
      setPlayers(players);
      setError(null);
      setConfirmModalOpen(false);
    } catch (error) {
      console.error('Failed to reset game:', error);
      setError(error instanceof Error ? error.message : 'Failed to reset game');
      setConfirmModalOpen(false);
    }
  };

  return (
    <main
      className={`flex w-full min-h-screen flex-col items-center justify-between p-2`}
    >
      <ConnectionModal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        onSubmit={handleHostnameSubmit}
      />
      <ConfirmationModal
        isOpen={confirmModalOpen}
        onClose={() => setConfirmModalOpen(false)}
        onConfirm={handleResetGame}
        title="ゲームをリセットしますか？"
        message="この操作は取り消せません。本当に実行してよろしいですか？"
        confirmText="リセット"
      />
      <ConfirmationModal
        isOpen={confirmAntennaDeleteModal.open}
        onClose={() => setConfirmAntennaDeleteModal({open: false, antennaId: null})}
        onConfirm={() => confirmAntennaDeleteModal.antennaId && handleDeleteAntenna(confirmAntennaDeleteModal.antennaId)}
        title="アンテナを削除しますか？"
        message="この操作は取り消せません。本当に実行してよろしいですか？"
        confirmText="削除"
      />
      <div className="flex-1 z-10 w-full max-w-5xl items-center justify-between font-mono text-sm bg-base-100">
        <div className="navbar navbar-center bg-base-100 w-full">
          <a className="btn btn-ghost navbar-start normal-case text-xl text-accent-content">Admin - RFID Poker</a>
          <div className="navbar-end">
            <button
              onClick={removeHostname}
              className="btn btn-primary normal-case"
            >Remove Endpoint</button>
          </div>
        </div>

        <h1 className="text-2xl font-bold mb-4">Hostname: {hostname}</h1>

        {error && (
          <div className="alert alert-error shadow-lg mb-4">
            <div>
              <svg xmlns="http://www.w3.org/2000/svg" className="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{error}</span>
            </div>
          </div>
        )}

        <div className="p-4">
          <h2 className="text-2xl font-bold mb-4">Antennas</h2>
          <div className="space-y-4 mb-8">
            {antennas?.map((antenna) => (
              <form 
                key={antenna.id} 
                className="flex items-center gap-4"
                onSubmit={(e) => {
                  e.preventDefault();
                  const formData = new FormData(e.currentTarget);
                  const newType = formData.get('antennaType') as AntennaTypeName;
                  handleAntennaTypeSubmit(antenna, newType);
                }}
              >
                <span className="w-24">ID {antenna.id}:</span>
                <span className="w-48">{antenna.device_id} / {antenna.pair_id}</span>
                <select 
                  name="antennaType"
                  defaultValue={antenna.antenna_type_name}
                  className="select select-bordered w-48"
                >
                  {ANTENNA_TYPES.map(type => (
                    <option key={type} value={type}>{type}</option>
                  ))}
                </select>
                <button type="submit" className="btn btn-primary">
                  Update Type
                </button>
                <button 
                  type="button"
                  onClick={() => setConfirmAntennaDeleteModal({open: true, antennaId: antenna.id})}
                  className="btn btn-error"
                >
                  Delete
                </button>
              </form>
            ))}
          </div>

          <h2 className="text-2xl font-bold mb-4">Players</h2>
          <div className="space-y-4">
            {players?.map((player, index) => (
              <div key={player.id}>
                <form 
                  className="flex items-center gap-4"
                  onSubmit={(e) => {
                    e.preventDefault();
                    const formData = new FormData(e.currentTarget);
                    const newName = formData.get('playerName') as string;
                    handlePlayerSubmit(player, newName);
                  }}
                >
                  <span className="w-24">Player {index + 1}:</span>
                  <span className="w-48">{player.device_id} / {player.pair_id}</span>
                  <input
                    type="text"
                    name="playerName"
                    defaultValue={player.name}
                    className="input input-bordered w-full max-w-xs"
                    placeholder={`Enter Player ${index + 1} name`}
                  />
                  <button type="submit" className="btn btn-primary">
                    Update
                  </button>
                </form>
                <div className="flex mt-2 ml-24 items-center">
                  {playerHands[player.id]?.error ? (
                    <span className="text-xl">{playerHands[player.id].error}</span>
                  ) : playerHands[player.id]?.is_muck ? (
                    <span className="text-xl">Mucked</span>
                  ) : (
                    playerHands[player.id]?.cards?.map((card, cardIndex) => (
                      <Card key={cardIndex} suit={card.suit} rank={card.rank} />
                    ))
                  )}
                  {playerHands[player.id]?.cards?.length > 0 && !playerHands[player.id]?.is_muck && (
                    <button 
                      onClick={() => handleMuckHand(player.id)}
                      className="btn btn-error ml-4"
                    >
                      Muck Hand
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>

          <div className="mt-8 border-t pt-4">
            <h2 className="text-2xl font-bold mb-4">Operations</h2>
            <div className="flex gap-4">
              <button
                onClick={() => setConfirmModalOpen(true)}
                className="btn btn-error"
              >
                Reset Game
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>
  )
}

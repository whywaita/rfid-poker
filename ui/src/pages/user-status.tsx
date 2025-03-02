import { useState, useEffect } from 'react';
import { ApiService, ApiPlayerType } from '@/services/api';
import ConnectionModal from '@/components/ConnectionModal';

export default function UserStatus() {
  const [modalOpen, setModalOpen] = useState(true);
  const [hostname, setHostname] = useState("");
  const [players, setPlayers] = useState<ApiPlayerType[]>([]);
  const [playerStatus, setPlayerStatus] = useState<{[key: number]: 'No Read' | 'Mucked' | 'Readed'}>({});
  const [error, setError] = useState<string | null>(null);
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
        checkPlayerHandStatus(player.id);
      });
    }
  }, [api, players]);

  useEffect(() => {
    if (api) {
      // Poll for updates every 2 seconds
      const interval = setInterval(() => {
        fetchData();
      }, 2000);
      return () => clearInterval(interval);
    }
  }, [api]);

  const handleHostnameSubmit = (inputHostname: string) => {
    setHostname(inputHostname);
    localStorage.setItem("hostname", inputHostname);
    const apiService = new ApiService(inputHostname);
    setApi(apiService);
    setModalOpen(false);
    fetchData();
  };

  const fetchData = async () => {
    if (!api) return;
    try {
      const playersResponse = await api.getPlayers();
      setPlayers(playersResponse.players);
      
      // Check status for each player
      playersResponse.players.forEach(player => {
        checkPlayerHandStatus(player.id);
      });
      
      setError(null);
    } catch (error) {
      console.error('Failed to fetch data:', error);
      setError(error instanceof Error ? error.message : 'Failed to fetch data');
    }
  };

  const checkPlayerHandStatus = async (playerId: number) => {
    if (!api) return;
    try {
      const response = await api.getPlayerHand(playerId);
      if (response.hand.is_muck) {
        setPlayerStatus(prev => ({
          ...prev,
          [playerId]: 'Mucked'
        }));
      } else if (response.hand.cards && response.hand.cards.length > 0) {
        setPlayerStatus(prev => ({
          ...prev,
          [playerId]: 'Readed'
        }));
      }
    } catch (error) {
      if (error instanceof Error && error.message === 'Not found') {
        setPlayerStatus(prev => ({
          ...prev,
          [playerId]: 'No Read'
        }));
        return;
      }
      console.error(`Failed to check status for player ${playerId}:`, error);
    }
  };

  function removeHostname() {
    localStorage.removeItem("hostname");
    setHostname("");
    setApi(null);
    setModalOpen(true);
  }

  return (
    <main className="flex w-full min-h-screen flex-col items-center justify-between p-2">
      <ConnectionModal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        onSubmit={handleHostnameSubmit}
      />
      
      <div className="flex-1 z-10 w-full max-w-5xl items-center justify-between font-mono text-sm bg-base-100">
        <div className="navbar navbar-center bg-base-100 w-full">
          <a className="btn btn-ghost navbar-start normal-case text-xl text-accent-content">User Status - RFID Poker</a>
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
          <h2 className="text-2xl font-bold mb-4">Players Status</h2>
          <div className="grid grid-cols-1 gap-4">
            {players.map((player) => (
              <div key={player.id} className="card bg-base-200 shadow-xl">
                <div className="card-body flex flex-row items-center justify-between">
                  <h2 className="card-title text-3xl">{player.name}</h2>
                  <StatusBadge status={playerStatus[player.id] || 'No Read'} />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </main>
  );
}

function StatusBadge({ status }: { status: 'No Read' | 'Mucked' | 'Readed' }) {
  let badgeClass = '';
  
  switch (status) {
    case 'No Read':
      badgeClass = 'badge-error';
      break;
    case 'Mucked':
      badgeClass = 'badge-warning';
      break;
    case 'Readed':
      badgeClass = 'badge-success';
      break;
  }
  
  return (
    <div className={`badge ${badgeClass} p-4 text-xl`}>
      {status}
    </div>
  );
} 
import Player, {PlayerType} from "@/components/Player";
import { useEffect, useState } from "react";

function Players({ hostname }:{hostname: string}) {
  const [players, setPlayers] = useState<PlayerType[]>([])

  useEffect(() => {
    if (!hostname) return;
    const ws = new WebSocket(`${hostname}/ws`);

    ws.onmessage = (event) => {
      try {
        const newData = JSON.parse(event.data);
        setPlayers(newData.players);
      } catch (e) {
        console.error("Error parsing JSON:", e);
      }
    }
    ws.onerror = (error) => {
      console.error("Websocket error:", error);
    }

    return () => {
      ws.close()
    }
  }, [hostname]);

  if (!players) { return <div></div> }

  return (
      <div className={"grid h-50"}>
        {players.map((player, index) => {
            return <Player player={player} key={index} />
          })}
      </div>
  )
}

function ConnectionModal({ isOpen, onClose, onSubmit }: { isOpen: boolean, onClose: () => void, onSubmit: (hostname: string) => void }) {
  const [hostname, setHostname] = useState("");
  const handleSubmit = () => {
    onSubmit(hostname);
    onClose();
  };
  if (!isOpen) return null;
  return (
      <div className="fixed inset-0 form-control items-center justify-center z-50">
        <div className="bg-primary-content p-4 rounded">
          <label className="label">
            <span className="label-text text-xl text-neutral">Endpoint (e.g. wss://192.0.2.1 )</span>
          </label>
          <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              className="input input-bordered p-3 text-primary-content"
          />
          <button onClick={handleSubmit} className="btn btn-primary ml-2">
            Set
          </button>
        </div>
      </div>
  );
}

export default function Home() {
  const [modalOpen, setModalOpen] = useState(true);
  const [hostname, setHostname] = useState("");

  useEffect(() => {
    const storedHostname = localStorage.getItem("hostname");
    if (storedHostname) {
      setHostname(storedHostname);
      setModalOpen(false);
    }
    return () => {
    };
  }, []);

  const handleHostnameSubmit = (inputHostname: string) => {
    setHostname(inputHostname);
    localStorage.setItem("hostname", inputHostname);
  };

  function removeHostname() {
    localStorage.removeItem("hostname");
    setHostname("");
    setModalOpen(true);
  }

  return (
    <main
      className={`flex w-full min-h-screen flex-col items-center justify-between p-2`}
    >
      <ConnectionModal
          isOpen={modalOpen}
          onClose={() => setModalOpen(false)}
          onSubmit={handleHostnameSubmit}
      />
      <div className="flex-1 z-10 w-full max-w-5xl items-center justify-between font-mono text-sm bg-base-100">
        <div className="navbar navbar-center bg-base-100 w-full">
          <a className="btn btn-ghost navbar-start normal-case text-xl text-neutral-50">RFID Poker</a>
          <div className="navbar-end">
            <button
                onClick={removeHostname}
                className="btn btn-primary normal-case"
            >Remove Endpoint</button>
          </div>
        </div>


        <Players hostname={hostname} />
      </div>
    </main>
  )
}

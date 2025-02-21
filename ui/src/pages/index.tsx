import Player, {PlayerType} from "@/components/Player";
import { useEffect, useState } from "react";
import Board  from "@/components/Board";
import {CardType} from "@/components/Cards";

import DOMPurify from 'dompurify';

function View({ hostname }:{hostname: string}) {
  const [players, setPlayers] = useState<PlayerType[]>([])
  const [board, setBoard] = useState<CardType[]>([])
  const [wsError, setWSError] = useState<string | null>(null);

  useEffect(() => {
    if (!hostname) return;
    const ws = new WebSocket(`${hostname}/ws`);

    ws.onmessage = (event) => {
      try {
        const newData = JSON.parse(event.data);
        setPlayers(newData.players);
        setBoard(newData.board);
      } catch (e) {
        console.error("Error parsing JSON:", e);
      }
    }
    ws.onerror = (error) => {
      console.error("Websocket error:", error);
      const sanitizedHostname = DOMPurify.sanitize(hostname);
      const errorMessage = error instanceof ErrorEvent && error.error
        ? `${error.error.message} (Hostname: ${sanitizedHostname})`
        : `Failed to connect to WebSocket at ${sanitizedHostname}`;
      setWSError(errorMessage);
    }

    return () => {
      ws.close()
    }
  }, [hostname]);

  if (wsError) {
    return <div role="alert" className="alert alert-error">
      <span>
        {wsError}
      </span>
    </div>
  }

  if (!players) { return <div></div> }

  return (
      <div className={"grid h-50"}>
        <Board cards={board} />
        {players.map((player, index) => {
            return <Player player={player} key={index} />
        })}
      </div>
  )
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

  return (
    <main
      className={`flex w-full min-h-screen flex-col items-center justify-between p-2`}
    >
      <div className="flex-1 z-10 w-full max-w-5xl items-center justify-between font-mono text-sm bg-base-100">
        <div className="navbar navbar-center bg-base-100 w-full">
          <a className="btn btn-ghost navbar-start normal-case text-xl text-accent-content">RFID Poker</a>
        </div>


        <View hostname={hostname} />
      </div>
    </main>
  )
}

import { PlayerType } from "@/components/Player";
import BroadcastPlayerBlock from "@/components/BroadcastPlayerBlock";

interface BroadcastUIProps {
  players: PlayerType[];
}

const BroadcastUI = ({ players }: BroadcastUIProps) => {
  if (!players || players.length === 0) return null;

  return (
    <div className="fixed bottom-6 left-6 z-50">
      <div className="flex flex-wrap gap-3 max-w-sm">
        {players.map((player, index) => (
          <BroadcastPlayerBlock key={index} player={player} />
        ))}
      </div>
    </div>
  );
};

export default BroadcastUI;
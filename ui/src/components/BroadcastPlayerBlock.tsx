import { PlayerType } from "@/components/Player";

interface BroadcastPlayerBlockProps {
  player: PlayerType;
}

const BroadcastPlayerBlock = ({ player }: BroadcastPlayerBlockProps) => {
  if (!player) return null;

  return (
    <div className="bg-black/80 text-white rounded-lg p-4 min-w-[180px] backdrop-blur-sm border border-gray-600">
      <div className="flex flex-col items-center space-y-2">
        <div className="w-16 h-16 bg-gray-700 rounded-full flex items-center justify-center">
          <span className="text-2xl font-bold text-gray-300">
            {player.name.charAt(0).toUpperCase()}
          </span>
        </div>
        
        <div className="text-center">
          <h3 className="text-lg font-semibold mb-1">{player.name}</h3>
          <div className="text-2xl font-bold text-green-400">
            {(player.equity * 100).toFixed(1)}%
          </div>
        </div>
      </div>
    </div>
  );
};

export default BroadcastPlayerBlock;
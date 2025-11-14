import { CardType } from "@/components/Cards";

export type BroadcastPlayerType = {
    name: string;
    hand: CardType[];
    equity: number;
    photoUrl?: string;
};

function ContentSuit(suit: CardType['suit']) {
    switch (suit) {
        case 'spades':
            return '♠';
        case 'hearts':
            return '♥';
        case 'diamonds':
            return '♦';
        case 'clubs':
            return '♣';
    }
}

function ContentRank(rank: CardType['rank']) {
    switch (rank) {
        case 'ace':
            return 'A';
        case 'jack':
            return 'J';
        case 'queen':
            return 'Q';
        case 'king':
            return 'K';
        default:
            return rank;
    }
}

function getCardColor(suit: CardType['suit']) {
    return suit === 'hearts' || suit === 'diamonds' ? 'text-red-600' : 'text-black';
}

const BroadcastPlayer = ({ player }: { player: BroadcastPlayerType }) => {
    if (!player || !player.name || !Array.isArray(player.hand)) {
        return <div></div>;
    }

    const equityPercentage = (player.equity * 100).toFixed(2);

    return (
        <div className="relative bg-black bg-opacity-90 rounded-lg p-4 flex items-center gap-4 min-w-[400px] h-[120px] shadow-lg overflow-visible">
            {/* Player Photo */}
            <img 
                src={player.photoUrl || "https://placehold.jp/3d4070/ffffff/500x500.png?text=Player"} 
                alt={player.name} 
                className="w-20 h-20 rounded-full object-cover border-2 border-white flex-shrink-0"
            />

            {/* Player Info */}
            <div className="flex flex-col items-left justify-center flex-1 h-full pt-5 pl-5">
                {/* Hand Cards - positioned above the box */}
                <div className="absolute top-[-20px] left-1/2 transform -translate-x-1/2 flex gap-2 z-10">
                    {player.hand.filter(card => card && card.suit && card.rank).map((card, index) => (
                        <div key={index} className="bg-white rounded-lg p-2 shadow-lg border-2 border-gray-300 flex flex-col items-center justify-center min-w-[50px] h-20">
                            <div className={`text-[40px] leading-none mb-0.5 ${getCardColor(card.suit)}`}>
                                {ContentSuit(card.suit)}
                            </div>
                            <div className={`text-[35px] font-bold leading-none ${getCardColor(card.suit)}`}>
                                {ContentRank(card.rank)}
                            </div>
                        </div>
                    ))}
                </div>

                {/* Player Name - positioned in the middle area between cards and bottom */}
                <div className="text-white text-[30px] font-bold text-left mt-8" style={{ textShadow: '2px 2px 4px rgba(0, 0, 0, 0.8)' }}>
                    {player.name}
                </div>
            </div>

            {/* Equity Display */}
            <div className="absolute right-0 top-0 flex items-center justify-end h-full min-w-20 gap-2.5 pr-0">
                <div className="text-yellow-400 text-[22px] font-bold text-center" style={{ textShadow: '0 2px 4px rgba(0, 0, 0, 0.8)' }}>
                    {equityPercentage}%
                </div>
                <div className="w-2 h-full bg-white bg-opacity-20 rounded relative">
                    <div 
                        className="absolute bottom-0 left-0 w-full bg-yellow-400 rounded shadow-lg"
                        style={{ 
                            height: `${player.equity * 100}%`,
                            boxShadow: '0 0 8px rgba(251, 191, 36, 0.6)'
                        }}
                    ></div>
                </div>
            </div>
        </div>
    );
};

export default BroadcastPlayer;
import Card, {CardType} from "@/components/Cards";

export type PlayerType = {
    name: string
    hand: CardType[]
    equity: number
};

const Player = ({ player }:{player: PlayerType}) => {
    if (!player) { return <div></div> }

    return (
        <div className={"flex w-full h-22 p-1 border-1 shadow-md bg-slate-50 items-center"}>
            <p className={"flex-auto text-center text-4xl w-1/4"}>{player.name}</p>
            <Card suit={player.hand[0].suit} rank={player.hand[0].rank} />
            <Card suit={player.hand[1].suit} rank={player.hand[1].rank} />
            <p className={"flex-auto text-center text-4xl w-1/4"}>{(player.equity*100).toFixed(2)}%</p>
        </div>
    );
};

export default Player;
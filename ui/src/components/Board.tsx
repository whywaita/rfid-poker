import Card, {CardType} from "@/components/Cards";

const Board = ({ cards }:{cards: CardType[]}) => {
    if (!cards) { return <div></div> }

    return (
        <div className={"flex w-full h-22 p-1 boarder-1 shadow-md bg-slate-50 items-center"}>
            <p className={"flex-auto text-center text-4xl"}>Board</p>
            {cards.map((card, index) => {
                return <Card suit={cards[index].suit} rank={cards[index].rank} key={index} />
            })}
        </div>
    );
};

export default Board;
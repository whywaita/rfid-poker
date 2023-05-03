export type Suit = 'spades' | 'hearts' | 'diamonds' | 'clubs';
export type Rank = 'ace' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | '10' | 'jack' | 'queen' | 'king';

export type CardType = {
    suit: Suit;
    rank: Rank;
};

function ContentSuit(suit: Suit) {
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

function ContentRank(rank: Rank) {
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

function color(suit: Suit) {
    return suit === 'hearts' || suit === 'diamonds' ? 'red-600' : 'black'
}

const Card = ({ suit, rank }:{suit: Suit, rank: Rank}) => {
    const contentClass = `flex-1 text-center text-5xl mt-3 ${color(suit) === 'red-600' ? 'text-red-600' : 'text-black'} text-opacity-100`

    return (
        <div className={"flex box-boarder bg-white h-20 w-1/4 p-1 mx-3 px-3 boarder-1 shadow-md"}>
            <p className={contentClass}>{ContentSuit(suit)}</p>
            <p className={contentClass}>{ContentRank(rank)}</p>
        </div>
    );
};

export default Card;
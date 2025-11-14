import { useState, useEffect } from 'react';
import BroadcastPlayer, { BroadcastPlayerType } from '@/components/BroadcastPlayer';
import { CardType } from '@/components/Cards';
import DOMPurify from 'dompurify';
import Head from 'next/head';
import { useRouter } from 'next/router';

// Convert PlayerType from WebSocket to BroadcastPlayerType
function convertToBroadcastPlayer(player: any): BroadcastPlayerType {
    if (!player || typeof player !== 'object') {
        throw new Error('Invalid player object received');
    }

    if (!player.name || typeof player.name !== 'string') {
        throw new Error('Player name is required and must be a string');
    }

    return {
        name: player.name,
        hand: player.hand || [],
        equity: typeof player.equity === 'number' && !isNaN(player.equity) ? player.equity : 0,
        photoUrl: `https://placehold.jp/3d4070/ffffff/500x500.png?text=${encodeURIComponent(player.name)}`
    };
}

function BroadcastView({ hostname }: { hostname: string }) {
    const [players, setPlayers] = useState<BroadcastPlayerType[]>([]);
    const [wsError, setWSError] = useState<string | null>(null);
    const [playerStates, setPlayerStates] = useState<Map<string, 'entering' | 'visible' | 'leaving'>>(new Map());

    useEffect(() => {
        if (!hostname) return;
        
        console.log("Connecting to WebSocket:", `${hostname}/ws`);
        const ws = new WebSocket(`${hostname}/ws`);

        ws.onopen = () => {
            console.log("WebSocket connected successfully");
            setWSError(null);
        };

        ws.onmessage = (event) => {
            try {
                const newData = JSON.parse(event.data);
                console.log("Received data:", newData);
                if (newData.players) {
                    const broadcastPlayers = newData.players.map(convertToBroadcastPlayer);
                    
                    // Track new and existing players
                    setPlayerStates(prevStates => {
                        const newStates = new Map(prevStates);
                        
                        // Mark new players as entering
                        broadcastPlayers.forEach(player => {
                            if (!prevStates.has(player.name)) {
                                newStates.set(player.name, 'entering');
                            }
                        });
                        
                        // Remove players that are no longer present
                        prevStates.forEach((state, playerName) => {
                            if (!broadcastPlayers.find(p => p.name === playerName)) {
                                newStates.delete(playerName);
                            }
                        });
                        
                        return newStates;
                    });
                    
                    setPlayers(broadcastPlayers);
                    
                    // After a short delay, mark entering players as visible
                    setTimeout(() => {
                        setPlayerStates(prevStates => {
                            const newStates = new Map(prevStates);
                            newStates.forEach((state, playerName) => {
                                if (state === 'entering') {
                                    newStates.set(playerName, 'visible');
                                }
                            });
                            return newStates;
                        });
                    }, 100);
                }
            } catch (e) {
                console.error("Error parsing JSON:", e);
            }
        };

        ws.onerror = (error) => {
            console.error("Websocket error:", error);
            const sanitizedHostname = DOMPurify.sanitize(hostname);
            const errorMessage = error instanceof ErrorEvent && error.error
                ? `${error.error.message} (Hostname: ${sanitizedHostname})`
                : `Failed to connect to WebSocket at ${sanitizedHostname}`;
            setWSError(errorMessage);
        };

        ws.onclose = (event) => {
            console.log("WebSocket closed:", event.code, event.reason);
        };

        return () => {
            ws.close();
        };
    }, [hostname]);

    if (wsError) {
        return (
            <div className="absolute top-5 left-5 bg-red-500 text-white p-4 rounded-lg">
                <span>{wsError}</span>
            </div>
        );
    }

    return (
        <div className="absolute bottom-5 left-5 flex flex-col gap-6">
            {players.map((player, index) => {
                const playerState = playerStates.get(player.name) || 'visible';
                
                return (
                    <div 
                        key={player.name} 
                        className={`${playerState === 'entering' ? 'animate-slideUp' : 'animate-slideUpComplete'}`}
                    >
                        <BroadcastPlayer player={player} />
                    </div>
                );
            })}
        </div>
    );
}

const BroadcastPage = () => {
    const [hostname, setHostname] = useState("");
    const router = useRouter();

    useEffect(() => {
        // Check for override-api-url query parameter first
        const overrideUrl = router.query['override-api-url'] as string;
        
        if (overrideUrl) {
            console.log("Using override API URL from query:", overrideUrl);
            setHostname(overrideUrl);
        } else {
            // Get hostname from localStorage (same as main page)
            const storedHostname = localStorage.getItem("hostname");
            console.log("Stored hostname:", storedHostname);
            if (storedHostname) {
                setHostname(storedHostname);
            } else {
                // Default hostname if not set
                console.log("No stored hostname, using default");
                setHostname("ws://localhost:8080");
            }
        }
    }, [router.query]);

    return (
        <>
            <Head>
                <style jsx>{`
                    @keyframes slideUp {
                        from {
                            transform: translateY(100%);
                            opacity: 0;
                        }
                        to {
                            transform: translateY(0);
                            opacity: 1;
                        }
                    }
                    
                    .animate-slideUp {
                        animation: slideUp 0.5s ease-out forwards;
                    }
                    
                    .animate-slideUpComplete {
                        transform: translateY(0);
                        opacity: 1;
                    }
                `}</style>
            </Head>
            <div className="min-h-screen bg-green-500 overflow-hidden">
                {/* Debug info - only show when debug=true */}
                {router.query.debug === 'true' && (
                    <div className="absolute top-5 right-5 bg-black bg-opacity-50 text-white p-2 rounded text-sm">
                        <div>Hostname: {hostname || "Not set"}</div>
                        <div className="text-xs opacity-75">
                            Source: {router.query['override-api-url'] ? 'URL Query' : 'localStorage'}
                        </div>
                    </div>
                )}
                
                {hostname ? (
                    <BroadcastView hostname={hostname} />
                ) : (
                    <div className="absolute top-5 left-5 bg-yellow-500 text-black p-4 rounded-lg">
                        <span>Please set hostname in the main page first or use ?override-api-url=ws://your-server</span>
                    </div>
                )}
            </div>
        </>
    );
};

export default BroadcastPage;
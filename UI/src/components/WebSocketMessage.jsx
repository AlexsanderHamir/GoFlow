import { useState, useEffect } from "react";

function WebSocketMessage() {
  const [message, setMessage] = useState("");
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const ws = new WebSocket("ws://localhost:8080/ws");

    ws.onopen = () => {
      console.log("Connected to WebSocket server");
      setIsConnected(true);
    };

    ws.onmessage = (event) => {
      console.log("Received message:", event.data);
      setMessage(event.data);
    };

    ws.onclose = () => {
      console.log("Disconnected from WebSocket server");
      setIsConnected(false);
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      setIsConnected(false);
    };

    // Cleanup on component unmount
    return () => {
      ws.close();
    };
  }, []); // Empty dependency array means this effect runs once on mount

  return (
    <div
      style={{
        position: "absolute",
        top: "10px",
        right: "10px",
        padding: "15px",
        backgroundColor: "rgba(255, 255, 255, 0.9)",
        borderRadius: "8px",
        boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
        zIndex: 1000,
      }}
    >
      <div style={{ marginBottom: "10px" }}>
        Connection Status: {isConnected ? "🟢 Connected" : "🔴 Disconnected"}
      </div>
      {message && (
        <div
          style={{
            padding: "10px",
            backgroundColor: "#f0f0f0",
            borderRadius: "4px",
            marginTop: "10px",
          }}
        >
          Message: {message}
        </div>
      )}
    </div>
  );
}

export default WebSocketMessage;

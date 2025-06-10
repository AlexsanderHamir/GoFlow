import React, { useEffect, useState } from "react";

function App() {
  const [messages, setMessages] = useState([]);

  useEffect(() => {
    const timer = setTimeout(() => {
      const ws = new WebSocket("ws://localhost:8080/ws");

      ws.onopen = () => console.log("WebSocket connected");

      ws.onmessage = async (event) => {
        try {
          const text = await event.data.text();
          const parsed = JSON.parse(text);
          console.log(parsed);
          setMessages((prev) => [...prev, parsed]);
        } catch (e) {
          console.error("Failed to parse message:", e);
        }
      };

      ws.onclose = () => console.log("WebSocket disconnected");
      ws.onerror = (error) => console.error("WebSocket error:", error);

      return () => ws.close();
    }, 1); // 1 millisecond delay

    return () => clearTimeout(timer);
  }, []);

  return (
    <div>
      <h1>Stage Setup Messages</h1>
      <ul>
        {messages.map((msg, index) => (
          <li key={index}>
            <strong>Type:</strong> {msg.type},<strong> StageName:</strong>{" "}
            {msg.stage_name},<strong> RoutineNum:</strong> {msg.routine_num},
            <strong> IsFinal:</strong> {String(msg.is_final)},
            <strong> IsGenerator:</strong> {String(msg.is_generator)}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default App;

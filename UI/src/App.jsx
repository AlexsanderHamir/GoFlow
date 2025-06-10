import React, { useEffect, useState, useCallback } from "react";
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
} from "reactflow";
import "reactflow/dist/style.css";

function App() {
  const [messages, setMessages] = useState(() => {
    // Load messages from localStorage on initial render
    const savedMessages = localStorage.getItem("stageSetupMessages");
    return savedMessages ? JSON.parse(savedMessages) : [];
  });

  // React Flow state
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  // Transform messages into nodes and edges
  const transformMessagesToFlow = useCallback((msgs) => {
    const newNodes = msgs.map((msg, index) => ({
      id: `stage-${msg.stage_name}-${msg.routine_num}`,
      data: {
        label: `${msg.stage_name}\nRoutine: ${msg.routine_num}\n${
          msg.is_generator ? "Generator" : "Stage"
        }`,
        ...msg,
      },
      position: { x: index * 250, y: index % 2 === 0 ? 100 : 250 },
      type: msg.is_generator ? "input" : "default",
    }));

    const newEdges = [];
    for (let i = 0; i < msgs.length - 1; i++) {
      if (!msgs[i].is_final) {
        newEdges.push({
          id: `e-${i}-${i + 1}`,
          source: newNodes[i].id,
          target: newNodes[i + 1].id,
          animated: true,
        });
      }
    }

    setNodes(newNodes);
    setEdges(newEdges);
  }, []);

  useEffect(() => {
    localStorage.removeItem("stageSetupMessages");
    localStorage.setItem("stageSetupMessages", JSON.stringify(messages));
    transformMessagesToFlow(messages);
  }, [messages, transformMessagesToFlow]);

  useEffect(() => {
    const timer = setTimeout(() => {
      const ws = new WebSocket("ws://localhost:8080/ws");

      ws.onopen = () => console.log("WebSocket connected");

      ws.onmessage = async (event) => {
        try {
          const text = await event.data.text();
          const parsed = JSON.parse(text);
          setMessages((prev) => [...prev, parsed]);
        } catch (e) {
          console.error("Failed to parse message:", e);
        }
      };

      ws.onclose = () => console.log("WebSocket disconnected");
      ws.onerror = (error) => console.error("WebSocket error:", error);

      return () => ws.close();
    }, 1);

    return () => clearTimeout(timer);
  }, []);

  return (
    <div style={{ width: "100vw", height: "100vh" }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        fitView
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}

export default App;

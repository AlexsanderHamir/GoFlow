import FlowExample from "./components/FlowExample";
import WebSocketMessage from "./components/WebSocketMessage";
import "./App.css";

function App() {
  return (
    <div className="App">
      <WebSocketMessage />
      <FlowExample />
    </div>
  );
}

export default App;

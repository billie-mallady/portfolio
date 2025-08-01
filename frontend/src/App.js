import React, { useState, useEffect, useRef } from 'react';
import './App.css';

const App = () => {
  const [history, setHistory] = useState([]);
  const [currentInput, setCurrentInput] = useState('');
  const [commandHistory, setCommandHistory] = useState([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const inputRef = useRef(null);
  const terminalBodyRef = useRef(null);
  
  // Rate limiting for chatbot
  const [chatbotRequests, setChatbotRequests] = useState([]);
  const [isRateLimited, setIsRateLimited] = useState(false);

const welcomeMessage = (
  <div>
    <pre style={{margin: 0}} className="ascii-title">
      <div className="billie-part" style={{color: '#F5A9B8'}}>
        ██████╗ ██╗██╗     ██╗     ██╗███████╗{'\n'}
        ██╔══██╗██║██║     ██║     ██║██╔════╝{'\n'}
        ██████╔╝██║██║     ██║     ██║█████╗  {'\n'}
        ██╔══██╗██║██║     ██║     ██║██╔══╝  {'\n'}
        ██████╔╝██║███████╗███████╗██║███████╗{'\n'}
        ╚═════╝ ╚═╝╚══════╝╚══════╝╚═╝╚══════╝
      </div>
      <div className="bot-part" style={{color: '#5BCEFA'}}>
        ██████╗  ██████╗ ████████╗{'\n'}
        ██╔══██╗██╔═══██╗╚══██╔══╝{'\n'}
        ██████╔╝██║   ██║   ██║   {'\n'}
        ██╔══██╗██║   ██║   ██║   {'\n'}
        ██████╔╝╚██████╔╝   ██║   {'\n'}
        ╚═════╝  ╚═════╝    ╚═╝   
      </div>
    </pre>
    <div style={{color: '#FFFFFF', marginTop: '10px'}}>
      Welcome to Billie Bot Terminal v1.0{'\n'}
      Type <span style={{color: '#5BCEFA'}}>'help'</span> to see available commands.{'\n'}
      Use command <span style={{color: '#F5A9B8'}}>'chatbot'</span> to ask the terminal questions in plain language...
    </div>
  </div>
);

  useEffect(() => {
    setHistory([{ type: 'output', content: welcomeMessage, isComponent: true }]);
    if (inputRef.current) {
      inputRef.current.focus();
    }
  }, []);

  useEffect(() => {
    if (terminalBodyRef.current) {
      terminalBodyRef.current.scrollTop = terminalBodyRef.current.scrollHeight;
    }
  }, [history]);

  // Rate limiting function
  // Allows max 10 requests in 5 minutes, and max 3 requests in 1 minute

  const checkRateLimit = () => {
    const now = Date.now();
    const fiveMinutesAgo = now - 5 * 60 * 1000; // 5 minutes
    const oneMinuteAgo = now - 60 * 1000; // 1 minute
    
    // Clean old requests
    const recentRequests = chatbotRequests.filter(timestamp => timestamp > fiveMinutesAgo);
    setChatbotRequests(recentRequests);
    
    // Check limits: max 10 requests per 5 minutes, max 3 per minute
    const requestsInLastMinute = recentRequests.filter(timestamp => timestamp > oneMinuteAgo);
    
    if (recentRequests.length >= 10) {
      setIsRateLimited(true);
      setTimeout(() => setIsRateLimited(false), 60000); // Cool down for 1 minute
      return false;
    }
    
    if (requestsInLastMinute.length >= 3) {
      setIsRateLimited(true);
      setTimeout(() => setIsRateLimited(false), 15000); // Cool down for 15 seconds
      return false;
    }
    
    return true;
  };

  // Input validation for chatbot - 
  // prevents long char inputs and suspicious patterns
  const validateChatbotInput = (input) => {
    // Check length (prevent extremely long inputs)
    if (input.length > 500) {
      return "Question too long. Please keep it under 500 characters.";
    }
    
    // Check for suspicious patterns
    const suspiciousPatterns = [
      /(.)\1{10,}/i, // Repeated characters
      /[^\w\s?!.,'"-]/g, // Only allow common punctuation
      /(hack|exploit|attack|inject|sql|script)/i // Common attack terms
    ];
    
    for (const pattern of suspiciousPatterns) {
      if (pattern.test(input)) {
        return "Invalid input detected. Please ask a normal question about the portfolio.";
      }
    }
    
    return null;
  };

  const executeCommand = async (command) => {
    const trimmedCommand = command.trim();
    if (!trimmedCommand) return;

    // Add command to history
    setHistory(prev => [...prev, { type: 'command', content: `$ ${trimmedCommand}` }]);
    setCommandHistory(prev => [...prev, trimmedCommand]);
    setHistoryIndex(-1);

    const parts = trimmedCommand.split(' ');
    const cmd = parts[0].toLowerCase();
    const args = parts.slice(1);

    let response = '';

    switch (cmd) {
      case 'help':
        response = 'Available commands:\n' +
          '  help                          - Show this help message\n' +
          '  chatbot <question>            - Ask BILLIEBOT a question about the portfolio\n' +
          '                                  (Limited: 3 requests/minute, 10 requests/5min)\n' +
          '  clear                         - Clear the terminal\n' +
          '  list <collection>             - List all documents in collection\n' +
          '  find <collection> <field> <value> - Find documents matching criteria\n' +
          '  count <collection>            - Count documents in collection\n' +
          '  show collections              - Show available collections\n' +
          '  exit                          - Exit the terminal\n\n' +
          'Examples:\n' +
          '  chatbot what projects use React?\n' +
          '  chatbot tell me about Billie\'s experience\n' +
          '  chatbot what skills does Billie have?\n' +
          '  list authors\n' +
          '  find projects category "Full Stack"\n' +
          '  count projects';
        break;

      case 'clear':
        setHistory([{ type: 'output', content: welcomeMessage, isComponent: true }]);
        return;

      case 'version':
        response = 'Portfolio Terminal v1.0\nBuilt with React and MongoDB\nGo Backend API';
        break;

      case 'list':
        if (args.length === 0) {
          response = 'Usage: list <collection>\nAvailable collections: authors, projects, resumes, education';
        } else {
          response = await handleListCommand(args[0]);
        }
        break;

      case 'find':
        if (args.length < 3) {
          response = 'Usage: find <collection> <field> <value>';
        } else {
          response = await handleFindCommand(args[0], args[1], args.slice(2).join(' '));
        }
        break;

      case 'count':
        if (args.length === 0) {
          response = 'Usage: count <collection>';
        } else {
          response = await handleCountCommand(args[0]);
        }
        break;

      case 'show':
        if (args[0] === 'collections') {
          response = 'Available collections:\n  • authors\n  • projects\n  • resumes\n  • education';
        } else {
          response = 'Usage: show collections';
        }
        break;

      case 'chatbot':
        if (args.length === 0) {
          response = 'Usage: chatbot <your question>\nExample: chatbot what projects use React?';
        } else {
          // Check rate limiting
          if (isRateLimited) {
            response = '🚫 Rate limit exceeded. Please wait before making another chatbot request.';
            break;
          }
          
          const question = args.join(' ');
          
          // Validate input
          const validationError = validateChatbotInput(question);
          if (validationError) {
            response = `🚫 ${validationError}`;
            break;
          }
          
          // Check rate limit before proceeding
          if (!checkRateLimit()) {
            response = '🚫 Too many requests. Please wait before asking another question.';
            break;
          }
          
          // Add timestamp to requests
          setChatbotRequests(prev => [...prev, Date.now()]);
          
          response = await handleChatbotCommand(question);
        }
        break;

      case 'exit':
        response = 'Thanks for using Portfolio Terminal!\nReload the page to restart.';
        break;

      default:
        response = `Command not found: ${cmd}\nType 'help' for available commands.`;
    }

    if (response) {
      setHistory(prev => [...prev, { type: 'output', content: response }]);
    }
  };

  const handleListCommand = async (collection) => {
    try {
      // Simulate API call - replace with actual fetch to your Go backend
      const response = await fetch(`/api/${collection}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      
      if (data.length === 0) {
        return `No documents found in collection '${collection}'`;
      }
      
      return `Found ${data.length} documents in '${collection}':\n${JSON.stringify(data, null, 2)}`;
    } catch (error) {
      return `Error connecting to backend: ${error.message}\nMake sure your Go server is running on port 8080`;
    }
  };

  const handleFindCommand = async (collection, field, value) => {
    try {
      const response = await fetch(`/api/${collection}/find?${field}=${encodeURIComponent(value)}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      
      if (data.length === 0) {
        return `No documents found matching ${field} = "${value}"`;
      }
      
      return `Found ${data.length} matching documents:\n${JSON.stringify(data, null, 2)}`;
    } catch (error) {
      return `Error: ${error.message}`;
    }
  };

  const handleCountCommand = async (collection) => {
    try {
      const response = await fetch(`/api/${collection}/count`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      
      return `Collection '${collection}' has ${data.count} documents`;
    } catch (error) {
      return `Error: ${error.message}`;
    }
  };

  const handleChatbotCommand = async (question) => {
    try {
      setHistory(prev => [...prev, { type: 'output', content: '🤖 BILLIEBOT is thinking...', thinking: true }]);
      
      const response = await fetch('/api/chatbot', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ query: question })
      });
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const data = await response.json();
      
      // Remove the "thinking" message
      setHistory(prev => {
        const idx = [...prev].reverse().findIndex(
          entry => entry.type === 'output' && entry.content === '🤖 BILLIEBOT is thinking...'
        );
        if (idx === -1) return prev;
        const removeIndex = prev.length - 1 - idx;
        return prev.filter((_, i) => i !== removeIndex);
      });
      
      return `🤖 BILLIEBOT:\n${data.response}`;
    } catch (error) {
      // Remove the "thinking" message on error too
      setHistory(prev => {
        const idx = [...prev].reverse().findIndex(
          entry => entry.type === 'output' && entry.content === '🤖 BILLIEBOT is thinking...'
        );
        if (idx === -1) return prev;
        const removeIndex = prev.length - 1 - idx;
        return prev.filter((_, i) => i !== removeIndex);
      });
      
      return `🤖 Chatbot Error: ${error.message}\nMake sure your Go server is running and OPENAI_API_KEY is configured.`;
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') {
      executeCommand(currentInput);
      setCurrentInput('');
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (commandHistory.length > 0) {
        const newIndex = historyIndex === -1 ? commandHistory.length - 1 : Math.max(0, historyIndex - 1);
        setHistoryIndex(newIndex);
        setCurrentInput(commandHistory[newIndex]);
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIndex !== -1) {
        const newIndex = historyIndex + 1;
        if (newIndex >= commandHistory.length) {
          setHistoryIndex(-1);
          setCurrentInput('');
        } else {
          setHistoryIndex(newIndex);
          setCurrentInput(commandHistory[newIndex]);
        }
      }
    }
  };

  return (
    <div className="terminal-container">
      <div className="terminal-header">
        <div className="terminal-buttons">
          <div className="terminal-button close"></div>
          <div className="terminal-button minimize"></div>
          <div className="terminal-button maximize"></div>
        </div>
        <div className="terminal-title">Portfolio Terminal - MongoDB Interface</div>
      </div>
      
      <div className="terminal-body" ref={terminalBodyRef}>
        <div className="terminal-output">
          {history.map((entry, index) => (
            <div key={index} className="terminal-line">
              {entry.type === 'command' ? (
                <span className="terminal-command">{entry.content}</span>
              ) : entry.type === 'error' ? (
                <span className="terminal-error">{entry.content}</span>
              ) : entry.isComponent ? (
                entry.content
              ) : entry.content.startsWith('🤖 BILLIEBOT:') ? (
                <div className="chatbot-response">{entry.content}</div>
              ) : (
                <pre className="terminal-result">{entry.content}</pre>
              )}
            </div>
          ))}
        </div>
        
        <div className="terminal-input-line">
          <span className="terminal-prompt-symbol">$</span>
            <span ></span>
          <input
            ref={inputRef}
            type="text"
            value={currentInput}
            onChange={(e) => setCurrentInput(e.target.value)}
            onKeyDown={handleKeyDown}
            className="terminal-input"
            autoFocus
          />
         
        </div>
      </div>
    </div>
  );
};

export default App;

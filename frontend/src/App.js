import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:3000';

function App() {
  const [bitcoins, setBitcoins] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);

  // Form state
  const [symbol, setSymbol] = useState('');
  const [price, setPrice] = useState('');
  const [isEditing, setIsEditing] = useState(false);
  const [editingSymbol, setEditingSymbol] = useState('');

  // Fetch bitcoins from API
  const fetchBitcoins = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.get(`${API_URL}/api/bitcoins`);
      setBitcoins(response.data);
    } catch (err) {
      setError('Failed to fetch bitcoins. Please check if the backend is running.');
      console.error('Error fetching bitcoins:', err);
    } finally {
      setLoading(false);
    }
  };

  // Initial fetch
  useEffect(() => {
    fetchBitcoins();
  }, []);

  // Handle form submission
  const handleSubmit = async (e) => {
    e.preventDefault();

    if (!symbol || !price) {
      setError('Please enter both symbol and price');
      return;
    }

    try {
      setError(null);
      setSuccess(null);

      await axios.post(`${API_URL}/api/bitcoins`, {
        symbol: symbol.toUpperCase(),
        price: parseInt(price)
      });

      const action = isEditing ? 'updated' : 'added';
      setSuccess(`Successfully ${action} ${symbol.toUpperCase()}`);
      setSymbol('');
      setPrice('');
      setIsEditing(false);
      setEditingSymbol('');

      // Refresh the list
      await fetchBitcoins();

      // Clear success message after 3 seconds
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError('Failed to add/update bitcoin. Please try again.');
      console.error('Error adding bitcoin:', err);
    }
  };

  // Handle edit
  const handleEdit = (bitcoin) => {
    setSymbol(bitcoin.symbol);
    setPrice(bitcoin.price.toString());
    setIsEditing(true);
    setEditingSymbol(bitcoin.symbol);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  // Handle cancel edit
  const handleCancelEdit = () => {
    setSymbol('');
    setPrice('');
    setIsEditing(false);
    setEditingSymbol('');
  };

  // Handle delete
  const handleDelete = async (symbolToDelete) => {
    if (!window.confirm(`Are you sure you want to delete ${symbolToDelete}?`)) {
      return;
    }

    try {
      setError(null);
      await axios.delete(`${API_URL}/api/bitcoins/${symbolToDelete}`);
      setSuccess(`Successfully deleted ${symbolToDelete}`);

      // Refresh the list
      await fetchBitcoins();

      // Clear success message after 3 seconds
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError('Failed to delete bitcoin. Please try again.');
      console.error('Error deleting bitcoin:', err);
    }
  };

  return (
    <div className="App">
      <div className="header">
        <h1>Bitcoin Cache Manager</h1>
        <p>Redis Cache with PostgreSQL Source of Truth</p>
      </div>

      {error && <div className="card error">{error}</div>}
      {success && <div className="card success">{success}</div>}

      <div className="card form-section">
        <h2>{isEditing ? `Edit ${editingSymbol}` : 'Add / Update Bitcoin'}</h2>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <input
              type="text"
              placeholder="Symbol (e.g., BTC)"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value)}
              maxLength={10}
              disabled={isEditing}
            />
            <input
              type="number"
              placeholder="Price (integer)"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
            />
            <button type="submit" className="btn btn-primary">
              {isEditing ? 'Update' : 'Save'}
            </button>
            {isEditing && (
              <button type="button" className="btn btn-secondary" onClick={handleCancelEdit}>
                Cancel
              </button>
            )}
          </div>
        </form>
      </div>

      <div className="card list-section">
        <h2>Bitcoin Rankings (by Price)</h2>

        {loading ? (
          <div className="loading">Loading...</div>
        ) : bitcoins.length === 0 ? (
          <div className="empty-state">
            <p>No bitcoins yet</p>
            <p>Add one using the form above</p>
          </div>
        ) : (
          <ul className="bitcoin-list">
            {bitcoins.map((bitcoin) => (
              <li key={bitcoin.symbol} className="bitcoin-item">
                <div className="bitcoin-rank">#{bitcoin.rank}</div>
                <div className="bitcoin-info">
                  <div className="bitcoin-symbol">{bitcoin.symbol}</div>
                  <div className="bitcoin-price">${bitcoin.price.toLocaleString()}</div>
                </div>
                <div className="bitcoin-actions">
                  <button
                    className="btn btn-secondary"
                    onClick={() => handleEdit(bitcoin)}
                  >
                    Edit
                  </button>
                  <button
                    className="btn btn-danger"
                    onClick={() => handleDelete(bitcoin.symbol)}
                  >
                    Delete
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

export default App;

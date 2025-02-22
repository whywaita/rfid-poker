import { useState } from 'react';

function ConnectionModal({ isOpen, onClose, onSubmit }: { isOpen: boolean, onClose: () => void, onSubmit: (hostname: string) => void }) {
    const [hostname, setHostname] = useState("");
    const handleSubmit = () => {
      onSubmit(hostname);
      onClose();
    };
    if (!isOpen) return null;
    return (
        <div className="fixed inset-0 form-control items-center justify-center z-50 rounded">
          <div className="bg-primary p-4 rounded">
            <label className="label">
              <span className="label-text text-xl text-primary-content">Endpoint (e.g. wss://192.0.2.1 )</span>
            </label>
            <input
                type="text"
                value={hostname}
                onChange={(e) => setHostname(e.target.value)}
                className="input input-bordered p-3"
            />
            <button onClick={handleSubmit} className="btn btn-primary ml-2">
              Set
            </button>
          </div>
        </div>
    );
  }

  export default ConnectionModal;
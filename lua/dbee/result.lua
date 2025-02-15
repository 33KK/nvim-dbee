---@alias result_config { mappings: table<string, mapping> }

-- Result is a wrapper around the go code
-- it is the central part of the plugin and manages connections.
-- almost all functions take the connection id as their argument.
---@class Result
---@field private ui Ui
---@field private handler Handler
local Result = {}

---@param ui Ui
---@param handler Handler
---@param opts? result_config
---@return Result
function Result:new(ui, handler, opts)
  opts = opts or {}

  if not handler then
    error("no Handler passed to Result")
  end
  if not ui then
    error("no Ui passed to Result")
  end

  -- class object
  local o = {
    ui = ui,
    handler = handler,
  }
  setmetatable(o, self)
  self.__index = self

  -- set keymaps
  o.ui:set_keymap(o:generate_keymap(opts.mappings))

  return o
end

---@private
---@param mappings table<string, mapping>
---@return keymap[]
function Result:generate_keymap(mappings)
  mappings = mappings or {}
  return {
    {
      action = function()
        self.handler:current_connection():page_next()
      end,
      mapping = mappings["page_next"],
    },
    {
      action = function()
        self.handler:current_connection():page_prev()
      end,
      mapping = mappings["page_prev"],
    },
  }
end

function Result:open()
  self.ui:open()
end

function Result:close()
  self.ui:close()
end

return Result

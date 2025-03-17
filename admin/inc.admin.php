<?php

class admin {
	static public $host = SOLPROXY_HOST;
	static public $last_error = false;
	
	static private function run($url)
	{
		self::$last_error = false;
		
		$data = file_get_contents($url);
		$data = json_decode($data, true);
		if (!is_array($data)) {
			$data = ['error'=>'Unknown error'];
		}
		if (isset($data['error']))
			self::$last_error = $data['error'];
		return $data;
	}
	
	static function nodeList() {
		self::$last_error = false;
		return self::run(self::$host."?action=evm_admin");
	}
	
	static function nodeAdd($config_json, $replace_node_id = false)
	{
		self::$last_error = false;
		$config_json = urlencode($config_json);
		return self::run(self::$host."?action=evm_admin_add&node={$config_json}&remove_id={$replace_node_id}");
	}
	
	static function nodeRemove($node_id = false)
	{
		self::$last_error = false;
		$data = self::run(self::$host."?action=evm_admin_remove&id={$node_id}");
		return $data;
	}
}